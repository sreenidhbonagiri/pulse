package monitoring

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"
)

func classifyRequestError(err error) string {
	if err == nil {
		return ""
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		err = urlErr.Err
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return "request timed out"
	}
	if errors.Is(err, context.Canceled) {
		return "request canceled"
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "request timed out"
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns lookup failed"
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return "connection failed"
	}

	return err.Error()
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
