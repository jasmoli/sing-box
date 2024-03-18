package route

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-dns"
	"github.com/sagernet/sing/common/cache"
	E "github.com/sagernet/sing/common/exceptions"
	F "github.com/sagernet/sing/common/format"
	M "github.com/sagernet/sing/common/metadata"

	mDNS "github.com/miekg/dns"
)

type DNSReverseMapping struct {
	cache *cache.LruCache[netip.Addr, string]
}

func NewDNSReverseMapping() *DNSReverseMapping {
	return &DNSReverseMapping{
		cache: cache.New[netip.Addr, string](),
	}
}

func (m *DNSReverseMapping) Save(address netip.Addr, domain string, ttl int) {
	m.cache.StoreWithExpire(address, domain, time.Now().Add(time.Duration(ttl)*time.Second))
}

func (m *DNSReverseMapping) Query(address netip.Addr) (string, bool) {
	domain, loaded := m.cache.Load(address)
	return domain, loaded
}

func (r *Router) matchDNS(ctx context.Context, allowFakeIP bool, index int) (context.Context, []dns.Transport, adapter.DNSRule, int, bool) {
	metadata := adapter.ContextFrom(ctx)
	if metadata == nil {
		panic("no context")
	}
	if index < len(r.dnsRules) {
		dnsRules := r.dnsRules
		if index != -1 {
			dnsRules = dnsRules[index+1:]
		}
		for currentRuleIndex, rule := range dnsRules {
			metadata.ResetRuleCache()
			if rule.Match(metadata) {
				var transports []dns.Transport
				var detours []string
				for _, detour := range rule.Servers() {
					transport, loaded := r.transportMap[detour]
					if !loaded {
						r.dnsLogger.ErrorContext(ctx, "transport not found: ", detour)
						continue
					}
					transports = append(transports, transport)
					detours = append(detours, detour)
				}
				if len(transports) == 0 {
					continue
				}
				_, isFakeIP := transports[0].(adapter.FakeIPTransport)
				if isFakeIP && !allowFakeIP {
					continue
				}
				ruleIndex := currentRuleIndex
				if index != -1 {
					ruleIndex += index + 1
				}
				detour := detours[0]
				if len(detours) > 1 {
					detour = "[" + strings.Join(detours, " ") + "]"
				}
				r.dnsLogger.DebugContext(ctx, "match[", currentRuleIndex, "] ", rule.String(), " => ", detour)
				if isFakeIP || rule.DisableCache() {
					ctx = dns.ContextWithDisableCache(ctx, true)
				}
				if rewriteTTL := rule.RewriteTTL(); rewriteTTL != nil {
					ctx = dns.ContextWithRewriteTTL(ctx, *rewriteTTL)
				}
				if clientSubnet := rule.ClientSubnet(); clientSubnet != nil {
					ctx = dns.ContextWithClientSubnet(ctx, *clientSubnet)
				}
				return ctx, transports, rule, ruleIndex, isFakeIP
			}
		}
	}
	return ctx, r.defaultTransports, nil, -1, false
}

func (r *Router) GetStrategy(transport dns.Transport) uint8 {
	if domainStrategy, dsLoaded := r.transportDomainStrategy[transport]; dsLoaded {
		return domainStrategy
	}
	return r.defaultDomainStrategy
}

type dnsRes struct {
	res *mDNS.Msg
	err error
	rej bool
}

func (r *Router) Exchange(ctx context.Context, message *mDNS.Msg) (*mDNS.Msg, error) {
	if len(message.Question) > 0 {
		r.dnsLogger.DebugContext(ctx, "exchange ", formatQuestion(message.Question[0].String()))
	}
	var (
		response *mDNS.Msg
		cached   bool
		isFakeIP bool
		err      error
	)
	defer func() {
		if err == nil && !isFakeIP && r.dnsReverseMapping != nil && len(message.Question) > 0 && response != nil && len(response.Answer) > 0 {
			for _, answer := range response.Answer {
				switch record := answer.(type) {
				case *mDNS.A:
					r.dnsReverseMapping.Save(M.AddrFromIP(record.A), fqdnToDomain(record.Hdr.Name), int(record.Hdr.Ttl))
				case *mDNS.AAAA:
					r.dnsReverseMapping.Save(M.AddrFromIP(record.AAAA), fqdnToDomain(record.Hdr.Name), int(record.Hdr.Ttl))
				}
			}
		}
	}()
	if response, cached = r.dnsClient.ExchangeCache(ctx, message); cached {
		return response, nil
	}
	ctx, metadata := adapter.AppendContext(ctx)
	if len(message.Question) > 0 {
		metadata.QueryType = message.Question[0].Qtype
		switch metadata.QueryType {
		case mDNS.TypeA:
			metadata.IPVersion = 4
		case mDNS.TypeAAAA:
			metadata.IPVersion = 6
		}
		metadata.Domain = fqdnToDomain(message.Question[0].Name)
	}

	ruleIndex := -1
	for {
		var (
			transports []dns.Transport
			rule       adapter.DNSRule
			dnsCtx     context.Context
		)
		dnsCtx, transports, rule, ruleIndex, isFakeIP = r.matchDNS(ctx, true, ruleIndex)
		resChan := make(chan dnsRes, len(transports))
		addressLimit := rule != nil && rule.WithAddressLimit() && isAddressQuery(message)
		var rejected bool
		for _, transport := range transports {
			go func(rawDNSCtx context.Context, transport dns.Transport) {
				var (
					res *mDNS.Msg
					err error
				)
				strategy := r.GetStrategy(transport)
				dnsCtx, cancel := context.WithTimeout(rawDNSCtx, C.DNSTimeout)
				if addressLimit {
					res, err = r.dnsClient.ExchangeWithResponseCheck(dnsCtx, transport, message, strategy, func(response *mDNS.Msg) bool {
						metadata.DestinationAddresses, _ = dns.MessageToAddresses(response)
						return rule.MatchAddressLimit(metadata)
					})
				} else {
					res, err = r.dnsClient.Exchange(dnsCtx, transport, message, strategy)
				}
				cancel()
				var rej bool
				if err != nil {
					if errors.Is(err, dns.ErrResponseRejectedCached) {
						rej = true
						r.dnsLogger.DebugContext(ctx, E.Cause(err, "response rejected for ", formatQuestion(message.Question[0].String())), " (cached)")
					} else if errors.Is(err, dns.ErrResponseRejected) {
						rej = true
						r.dnsLogger.DebugContext(ctx, E.Cause(err, "response rejected for ", formatQuestion(message.Question[0].String())))
					} else if len(message.Question) > 0 {
						r.dnsLogger.ErrorContext(ctx, E.Cause(err, "exchange failed for ", formatQuestion(message.Question[0].String())))
					} else {
						r.dnsLogger.ErrorContext(ctx, E.Cause(err, "exchange failed for <empty query>"))
					}
				}
				resChan <- dnsRes{
					res: res,
					err: err,
					rej: rej,
				}
			}(dnsCtx, transport)
		}
		res := <-resChan
		response = res.res
		err = res.err
		rejected = res.rej
		if addressLimit && rejected {
			continue
		}
		break
	}
	return response, err
}

type dnsAddr struct {
	addrs []netip.Addr
	err   error
}

func (r *Router) Lookup(ctx context.Context, domain string, strategy dns.DomainStrategy) ([]netip.Addr, error) {
	if responseAddrs, cached := r.dnsClient.LookupCache(ctx, domain, strategy); cached {
		return responseAddrs, nil
	}
	r.dnsLogger.DebugContext(ctx, "lookup domain ", domain)
	ctx, metadata := adapter.AppendContext(ctx)
	metadata.Domain = domain
	defer metadata.ResetRuleCache()
	var (
		responseAddrs []netip.Addr
		err           error
	)
	ruleIndex := -1
	for {
		var (
			transports []dns.Transport
			rule       adapter.DNSRule
			dnsCtx     context.Context
		)
		metadata.ResetRuleCache()
		metadata.DestinationAddresses = nil
		dnsCtx, transports, rule, ruleIndex, _ = r.matchDNS(ctx, false, ruleIndex)
		resChan := make(chan dnsAddr, len(transports))
		addressLimit := rule != nil && rule.WithAddressLimit()
		for _, transport := range transports {
			go func(rawDNSCtx context.Context, transport dns.Transport) {
				var (
					addrs []netip.Addr
					err   error
				)
				strategy := r.GetStrategy(transport)
				dnsCtx, cancel := context.WithTimeout(rawDNSCtx, C.DNSTimeout)
				if addressLimit {
					addrs, err = r.dnsClient.LookupWithResponseCheck(dnsCtx, transport, domain, strategy, func(addrs []netip.Addr) bool {
						metadata.DestinationAddresses = addrs
						return rule.MatchAddressLimit(metadata)
					})
				} else {
					addrs, err = r.dnsClient.Lookup(dnsCtx, transport, domain, strategy)
				}
				cancel()
				if err != nil {
					if errors.Is(err, dns.ErrResponseRejectedCached) {
						r.dnsLogger.DebugContext(ctx, "response rejected for ", domain, " (cached)")
					} else if errors.Is(err, dns.ErrResponseRejected) {
						r.dnsLogger.DebugContext(ctx, "response rejected for ", domain)
					} else {
						r.dnsLogger.ErrorContext(ctx, E.Cause(err, "lookup failed for ", domain))
					}
				} else if len(addrs) == 0 {
					r.dnsLogger.ErrorContext(ctx, "lookup failed for ", domain, ": empty result")
					err = dns.RCodeNameError
				} else {
					r.dnsLogger.DebugContext(ctx, "lookup succeed for ", domain, ": ", strings.Join(F.MapToString(addrs), " "))
				}
				resChan <- dnsAddr{
					addrs: addrs,
					err:   err,
				}
			}(dnsCtx, transport)
		}
		dnsAddrs := <-resChan
		responseAddrs = dnsAddrs.addrs
		err = dnsAddrs.err
		if !addressLimit || err == nil {
			break
		}
	}
	if len(responseAddrs) > 0 {
		r.dnsLogger.InfoContext(ctx, "finally lookup succeed for ", domain, ": ", strings.Join(F.MapToString(responseAddrs), " "))
	}
	return responseAddrs, err
}

func (r *Router) LookupDefault(ctx context.Context, domain string) ([]netip.Addr, error) {
	return r.Lookup(ctx, domain, dns.DomainStrategyAsIS)
}

func (r *Router) ClearDNSCache() {
	r.dnsClient.ClearCache()
	if r.platformInterface != nil {
		r.platformInterface.ClearDNSCache()
	}
}

func isAddressQuery(message *mDNS.Msg) bool {
	for _, question := range message.Question {
		if question.Qtype == mDNS.TypeA || question.Qtype == mDNS.TypeAAAA {
			return true
		}
	}
	return false
}

func fqdnToDomain(fqdn string) string {
	if mDNS.IsFqdn(fqdn) {
		return fqdn[:len(fqdn)-1]
	}
	return fqdn
}

func formatQuestion(string string) string {
	if strings.HasPrefix(string, ";") {
		string = string[1:]
	}
	string = strings.ReplaceAll(string, "\t", " ")
	for strings.Contains(string, "  ") {
		string = strings.ReplaceAll(string, "  ", " ")
	}
	return string
}
