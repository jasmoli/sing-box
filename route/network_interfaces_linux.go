package route

import (
	"github.com/sagernet/netlink"
	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing/common/control"
)

func (r *NetworkManager) updateSystemInterfaces() {
	links, err := netlink.LinkList()
	if err != nil {
		return
	}
	wifiConnected := r.WIFIState().SSID != ""
	interfaces := make([]adapter.NetworkInterface, 0, len(links))
	for _, link := range links {
		attrs := link.Attrs()
		if attrs == nil {
			continue
		}
		interfaces = append(interfaces, adapter.NetworkInterface{
			Interface: control.Interface{
				Index: attrs.Index,
				MTU:   attrs.MTU,
				Name:  attrs.Name,
				Flags: attrs.Flags,
			},
			Type: systemInterfaceType(link, wifiConnected),
		})
	}
	r.networkInterfaces.Store(interfaces)
}

func systemInterfaceType(link netlink.Link, wifiConnected bool) C.InterfaceType {
	if linkType := link.Type(); linkType != "" && linkType != "device" {
		return C.InterfaceTypeCellular
	}
	attrs := link.Attrs()
	if attrs != nil && attrs.EncapType == "ether" {
		if wifiConnected {
			return C.InterfaceTypeWIFI
		}
		return C.InterfaceTypeEthernet
	}
	return C.InterfaceTypeOther
}
