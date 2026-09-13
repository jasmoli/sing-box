//go:build !linux && !android

package route

func (r *NetworkManager) updateSystemInterfaces() {}
