package module

import (
	"net"
	"sync"
)

//FilterOptions for IPFilter. Allow/Block setting
type FilterOptions struct {
	//explicity allowed IPs
	AllowedIPs     []string            `json:"allowed"`
	BlockedIPs     []string            `json:"blocked"`
	URLPath        string              `json:"urlPath"`
	URLParam       string              `json:"urlParam"`
	AuthorizedIPs  map[string][]string `json:"authorized"`
	BlockByDefault bool                `json:"blockedDefault"`
}

//IPFilter filter struct
type IPFilter struct {
	opts           FilterOptions
	mut            sync.RWMutex
	defaultAllowed bool
	allowedIPs     map[string]bool
	urlPath        string
	urlParam       string
	blockedIPs     map[string]bool
	authorizedIPs  map[string][]string
}

//New new ipfilter
func New(opts FilterOptions) *IPFilter {

	f := &IPFilter{
		opts:           opts,
		allowedIPs:     map[string]bool{},
		blockedIPs:     map[string]bool{},
		authorizedIPs:  map[string][]string{},
		defaultAllowed: !opts.BlockByDefault,
	}
	f.urlParam = opts.URLParam
	f.urlPath = opts.URLPath
	for _, ip := range opts.AllowedIPs {
		f.allowIP(ip)
	}
	for _, ip := range opts.BlockedIPs {
		f.blockIP(ip)
	}
	for k, v := range opts.AuthorizedIPs {
		for _, ip := range v {
			f.authorizeIP(ip, k)
		}

	}
	return f
}

//allowIP  settting allow ip address
func (f *IPFilter) allowIP(ip string) bool {
	if ip := net.ParseIP(ip); ip != nil {
		f.mut.Lock()
		f.allowedIPs[ip.String()] = true
		f.mut.Unlock()
		return true
	}
	return false
}

//blockIP setting block ip address
func (f *IPFilter) blockIP(ip string) bool {
	if ip := net.ParseIP(ip); ip != nil {
		f.mut.Lock()
		f.blockedIPs[ip.String()] = true
		f.mut.Unlock()
		return true
	}
	return false
}

//authorizeIP settting service authorized ip address
func (f *IPFilter) authorizeIP(ip string, identity string) bool {

	if ip := net.ParseIP(ip); ip != nil && len(identity) > 0 {
		f.mut.Lock()
		val, ok := f.authorizedIPs[identity]
		if ok {
			val = append(val, ip.String())
			f.authorizedIPs[identity] = val
		} else {
			val = make([]string, 0)
			val = append(val, ip.String())
			f.authorizedIPs[identity] = val
		}
		return true
	}
	return false
}

func (f *IPFilter) Allowed(ip string) bool {
	if ip == "" {
		return f.defaultAllowed
	}
	allowed, ok := f.allowedIPs[ip]
	if ok {
		return allowed
	}
	blocked, ok := f.blockedIPs[ip]
	if ok {
		return !blocked
	}
	return f.defaultAllowed
}

// func (f *IPFilter) Blocked(ip string) bool {
// 	if ip == "" {
// 		return false
// 	}
// 	blocked, ok := f.blockedIPs[ip]
// 	if ok {
// 		return blocked
// 	}
// 	return false
// }

func (f *IPFilter) Authorized(ip string, param string) bool {
	if len(param) == 0 {
		return false
	}

	if ips, ok := f.authorizedIPs[param]; ok {
		for _, val := range ips {
			if ip == val {
				return true
			}
		}
	}

	return false
}
