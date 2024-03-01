package rest

import "net/url"

// Service post
func Service(host string, path string, req any, resp any) (code int, err error) {

	var p string
	if len(host) == 0 {
		code = 404
		return
	}
	if len(path) > 0 {
		p, err = url.JoinPath(host, path)
		if err != nil {
			return
		}
	} else {
		p = host
	}
	code, err = New().PostJson(p, req, resp)
	return
}

// Fetch get
func Fetch(host string, path string, req any, resp any) (code int, err error) {
	var p string
	if len(host) == 0 {
		code = 404
		return
	}
	if len(path) > 0 {
		p, err = url.JoinPath(host, path)
		if err != nil {
			return
		}
	} else {
		p = host
	}

	code, err = New().GetJson(p, req, resp)
	return
}
