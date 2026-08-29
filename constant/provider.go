package constant

const (
	ProviderTypeLocal  = "local"
	ProviderTypeRemote = "remote"
	ProviderTypeInline = "inline"
)

func ProviderDisplayName(providerType string) string {
	switch providerType {
	case ProviderTypeLocal:
		return "File"
	case ProviderTypeRemote:
		return "HTTP"
	case ProviderTypeInline:
		return "Inline"
	default:
		return "Unknown"
	}
}
