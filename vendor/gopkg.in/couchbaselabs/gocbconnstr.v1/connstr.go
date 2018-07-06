package gocbconnstr

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
)

const (
	DefaultHttpPort    = 8091
	DefaultMemdPort    = 11210
	DefaultSslMemdPort = 11207
)

type Address struct {
	Host string
	Port int
}

type ConnSpec struct {
	Scheme    string
	Addresses []Address
	Bucket    string
	Options   map[string][]string
}

func (spec ConnSpec) GetOption(name string) []string {
	if opt, ok := spec.Options[name]; ok {
		return opt
	}
	return nil
}

func (spec ConnSpec) GetOptionString(name string) string {
	opts := spec.GetOption(name)
	if len(opts) > 0 {
		return opts[0]
	}
	return ""
}

func Parse(connStr string) (out ConnSpec, err error) {
	partMatcher := regexp.MustCompile(`((.*):\/\/)?(([^\/?:]*)(:([^\/?:@]*))?@)?([^\/?]*)(\/([^\?]*))?(\?(.*))?`)
	hostMatcher := regexp.MustCompile(`((\[[^\]]+\]+)|([^;\,\:]+))(:([0-9]*))?(;\,)?`)
	parts := partMatcher.FindStringSubmatch(connStr)

	if parts[2] != "" {
		out.Scheme = parts[2]

		switch out.Scheme {
		case "couchbase":
		case "couchbases":
		case "http":
		default:
			err = errors.New("bad scheme")
			return
		}
	}

	if parts[7] != "" {
		hosts := hostMatcher.FindAllStringSubmatch(parts[7], -1)
		for _, hostInfo := range hosts {
			address := Address{
				Host: hostInfo[1],
				Port: -1,
			}

			if hostInfo[5] != "" {
				address.Port, err = strconv.Atoi(hostInfo[5])
				if err != nil {
					return
				}
			}

			out.Addresses = append(out.Addresses, address)
		}
	}

	if parts[9] != "" {
		out.Bucket, err = url.QueryUnescape(parts[9])
		if err != nil {
			return
		}
	}

	if parts[11] != "" {
		out.Options, err = url.ParseQuery(parts[11])
		if err != nil {
			return
		}
	}

	return
}

func (spec ConnSpec) String() string {
	var out string

	if spec.Scheme != "" {
		out += fmt.Sprintf("%s://", spec.Scheme)
	}

	for i, address := range spec.Addresses {
		if i > 0 {
			out += ","
		}

		if address.Port >= 0 {
			out += fmt.Sprintf("%s:%d", address.Host, address.Port)
		} else {
			out += address.Host
		}
	}

	if spec.Bucket != "" {
		out += "/"
		out += spec.Bucket
	}

	urlOptions := url.Values(spec.Options)
	if len(urlOptions) > 0 {
		out += "?" + urlOptions.Encode()
	}

	return out
}

type ResolvedConnSpec struct {
	UseSsl    bool
	MemdHosts []Address
	HttpHosts []Address
	Bucket    string
	Options   map[string][]string
}

func Resolve(connSpec ConnSpec) (out ResolvedConnSpec, err error) {
	defaultPort := 0
	hasExplicitScheme := false
	isHttpScheme := false
	useSsl := false

	switch connSpec.Scheme {
	case "couchbase":
		defaultPort = DefaultMemdPort
		hasExplicitScheme = true
		isHttpScheme = false
		useSsl = false
	case "couchbases":
		defaultPort = DefaultSslMemdPort
		hasExplicitScheme = true
		isHttpScheme = false
		useSsl = true
	case "http":
		defaultPort = DefaultHttpPort
		hasExplicitScheme = true
		isHttpScheme = true
		useSsl = false
	case "":
		defaultPort = DefaultHttpPort
		hasExplicitScheme = false
		isHttpScheme = true
		useSsl = false
	default:
		err = errors.New("bad scheme")
		return
	}

	var srvRecords []*net.SRV
	if !isHttpScheme && len(connSpec.Addresses) == 1 && connSpec.Addresses[0].Port == -1 {
		srvHostname := connSpec.Addresses[0].Host
		_, addrs, err := net.LookupSRV(connSpec.Scheme, "tcp", srvHostname)
		if err == nil && len(addrs) > 0 {
			srvRecords = addrs
		}
	}

	if srvRecords != nil {
		for _, srv := range srvRecords {
			out.MemdHosts = append(out.MemdHosts, Address{
				Host: srv.Target,
				Port: int(srv.Port),
			})
		}
	} else if len(connSpec.Addresses) == 0 {
		if useSsl {
			out.MemdHosts = append(out.MemdHosts, Address{
				Host: "127.0.0.1",
				Port: DefaultSslMemdPort,
			})
		} else {
			out.MemdHosts = append(out.MemdHosts, Address{
				Host: "127.0.0.1",
				Port: DefaultMemdPort,
			})
			out.HttpHosts = append(out.HttpHosts, Address{
				Host: "127.0.0.1",
				Port: DefaultHttpPort,
			})
		}
	} else {
		for _, address := range connSpec.Addresses {
			hasExplicitPort := address.Port > 0

			if !hasExplicitScheme && hasExplicitPort && address.Port != defaultPort {
				err = errors.New("Ambiguous port without scheme")
				return
			}

			if hasExplicitScheme && !isHttpScheme && address.Port == DefaultHttpPort {
				err = errors.New("couchbase://host:8091 not supported for couchbase:// scheme. Use couchbase://host")
				return
			}

			if address.Port <= 0 || address.Port == defaultPort || address.Port == DefaultHttpPort {
				if useSsl {
					out.MemdHosts = append(out.MemdHosts, Address{
						Host: address.Host,
						Port: DefaultSslMemdPort,
					})
				} else {
					out.MemdHosts = append(out.MemdHosts, Address{
						Host: address.Host,
						Port: DefaultMemdPort,
					})
					out.HttpHosts = append(out.HttpHosts, Address{
						Host: address.Host,
						Port: DefaultHttpPort,
					})
				}
			} else {
				if !isHttpScheme {
					out.MemdHosts = append(out.MemdHosts, Address{
						Host: address.Host,
						Port: address.Port,
					})
				} else {
					out.HttpHosts = append(out.HttpHosts, Address{
						Host: address.Host,
						Port: address.Port,
					})
				}
			}
		}
	}

	out.UseSsl = useSsl
	out.Bucket = connSpec.Bucket
	out.Options = connSpec.Options
	return
}
