package clashapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/badjson"
	F "github.com/sagernet/sing/common/format"
)

func proxyProviderRouter(server *Server, router adapter.Router) http.Handler {
	r := chi.NewRouter()
	r.Get("/", getProviders(server, router))

	r.Route("/{name}", func(r chi.Router) {
		r.Use(parseProviderName, findProviderByName(router))
		r.Get("/", getProvider(server))
		r.Put("/", updateProvider(server, router))
		r.Get("/healthcheck", healthCheckProvider(server))
	})
	return r
}

func providerInfo(server *Server, provider adapter.OutboundProvider) *badjson.JSONObject {
	var info badjson.JSONObject
	proxyArray := make(badjson.JSONArray, 0)
	for _, outbound := range provider.Outbounds() {
		proxyArray = append(proxyArray, proxyInfo(server, outbound))
	}
	info.Put("name", provider.Tag())
	info.Put("type", "Proxy")
	info.Put("vehicleType", strings.ToUpper(provider.Type()))
	info.Put("subscriptionInfo", provider.SubscriptionInfo())
	info.Put("updatedAt", provider.UpdateTime().Format("2006-01-02T15:04:05.999999999-07:00"))
	info.Put("proxies", &proxyArray)
	return &info
}

func getProviders(server *Server, router adapter.Router) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var providerMap badjson.JSONObject
		for i, provider := range router.OutboundProviders() {
			var tag string
			if provider.Tag() == "" {
				tag = F.ToString(i)
			} else {
				tag = provider.Tag()
			}
			providerMap.Put(tag, providerInfo(server, provider))
		}
		var responseMap badjson.JSONObject
		responseMap.Put("providers", &providerMap)
		response, err := responseMap.MarshalJSON()
		if err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, newError(err.Error()))
			return
		}
		w.Write(response)
	}
}

func getProvider(server *Server) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		provider := r.Context().Value(CtxKeyProvider).(adapter.OutboundProvider)
		response, err := providerInfo(server, provider).MarshalJSON()
		if err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, newError(err.Error()))
			return
		}
		w.Write(response)
	}
}

func updateProvider(server *Server, router adapter.Router) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		provider := r.Context().Value(CtxKeyProvider).(adapter.OutboundProvider)
		err := provider.UpdateProvider(server.ctx, router, true)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, err)
			return
		}
		render.NoContent(w, r)
	}
}

func healthCheckProvider(server *Server) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		provider := r.Context().Value(CtxKeyProvider).(adapter.OutboundProvider)

		query := r.URL.Query()
		link := query.Get("url")
		timeout := int64(5000)

		ctx, cancel := context.WithTimeout(r.Context(), time.Millisecond*time.Duration(timeout))
		defer cancel()

		result, _ := provider.Healthcheck(ctx, link, true)
		render.JSON(w, r, result)
	}
}

func parseProviderName(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := getEscapeParam(r, "name")
		ctx := context.WithValue(r.Context(), CtxKeyProviderName, name)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func findProviderByName(router adapter.Router) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			name := r.Context().Value(CtxKeyProviderName).(string)
			provider, exist := router.OutboundProvider(name)
			if !exist {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, ErrNotFound)
				return
			}
			ctx := context.WithValue(r.Context(), CtxKeyProvider, provider)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
