package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/darahayes/go-boom"
	"github.com/suchimauz/proxy-http-server-client/internal/config"
	"github.com/suchimauz/proxy-http-server-client/pkg/logger"

	netProxy "golang.org/x/net/proxy"
)

type (
	Proxy struct {
		ProxyType string `json:"type"`
		Host      string `json:"host"`
		Port      int    `json:"port"`
		Username  string `json:"username"`
		Password  string `json:"password"`
	}
	HttpRequest struct {
		Url     string            `json:"url"`
		Method  string            `json:"method"`
		Body    json.RawMessage   `json:"body"`
		Headers map[string]string `json:"headers"`
		Params  map[string]string `json:"params"`
		Proxy   *Proxy            `json:"proxy"`
	}
	HttpResponse struct {
		Request  *HttpRequest    `json:"request"`
		Response json.RawMessage `json:"response"`
	}
)

var (
	fmtProxyMax            = "%s://%s:%s@%s:%s"
	fmtProxyMaxWithoutAuth = "%s://%s:%s"
	fmtProxyMin            = "%s://%s"
	fmtProxyMinWithAuth    = "%s://%s:%s@%s"
)

func calcProxy(proxy *Proxy) (*http.Transport, error) {
	var dialer netProxy.Dialer
	var transport *http.Transport
	var err error
	var proxyURL *url.URL
	var proxyMax string
	var proxyMaxWithoutAuth string
	var proxyMin string
	var proxyMinWithAuth string

	if proxy != nil {
		proxyMax = fmt.Sprintf(fmtProxyMax, proxy.ProxyType, proxy.Username, proxy.Password, proxy.Host, strconv.Itoa(proxy.Port))
		proxyMaxWithoutAuth = fmt.Sprintf(fmtProxyMaxWithoutAuth, proxy.ProxyType, proxy.Host, strconv.Itoa(proxy.Port))
		proxyMin = fmt.Sprintf(fmtProxyMin, proxy.ProxyType, proxy.Host)
		proxyMinWithAuth = fmt.Sprintf(fmtProxyMinWithAuth, proxy.ProxyType, proxy.Username, proxy.Password, proxy.Host)

		if proxy.Username != "" && proxy.Password != "" {
			if proxy.Port == 0 {
				if proxyURL, err = url.Parse(proxyMinWithAuth); err != nil {
					return nil, err
				}
			} else {
				if proxyURL, err = url.Parse(proxyMax); err != nil {
					return nil, err
				}
			}
		} else {
			if proxy.Port == 0 {
				if proxyURL, err = url.Parse(proxyMin); err != nil {
					return nil, err
				}
			} else {
				if proxyURL, err = url.Parse(proxyMaxWithoutAuth); err != nil {
					return nil, err
				}
			}
		}

		switch proxyType := proxy.ProxyType; proxyType {
		case "http":
			transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		case "socks5":
			if dialer, err = netProxy.FromURL(proxyURL, netProxy.Direct); err != nil {
				return nil, err
			}
			transport = &http.Transport{
				Dial: dialer.Dial,
			}
		default:
			return nil, fmt.Errorf("Unsupported proxy type: %s", proxyType)
		}

		return transport, nil
	}
	return nil, nil
}

func callRequest(request *HttpRequest) ([]byte, error) {
	var httpClient *http.Client

	transport, err := calcProxy(request.Proxy)
	if err != nil {
		return nil, err
	}

	if transport != nil {
		httpClient = &http.Client{
			Transport: transport,
		}
	} else {
		httpClient = &http.Client{}
	}

	requestUrl, err := url.Parse(request.Url)
	if err != nil {
		return nil, err
	}

	requestParams := url.Values{}
	for k, v := range request.Params {
		requestParams.Add(k, v)
	}

	requestUrl.RawQuery = requestParams.Encode()

	var requestMethod string

	switch m := strings.ToLower(request.Method); m {
	case "get":
		requestMethod = http.MethodGet
	case "post":
		requestMethod = http.MethodPost
	case "put":
		requestMethod = http.MethodPut
	case "delete":
		requestMethod = http.MethodDelete
	default:
		return nil, fmt.Errorf("Unsupported request method: %s", m)
	}

	var req *http.Request
	if len(request.Body) > 0 {
		req, err = http.NewRequest(requestMethod, requestUrl.String(), bytes.NewReader(request.Body))
	} else {
		req, err = http.NewRequest(requestMethod, requestUrl.String(), nil)
	}
	if err != nil {
		return nil, err
	}

	for k, v := range request.Headers {
		req.Header.Add(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	if err = resp.Write(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func validateProxifyBody(body *HttpRequest) error {
	if body.Url == "" {
		return errors.New("body.url are required")
	} else if body.Method == "" {
		return errors.New("body.method are required")
	}
	return nil
}

func proxifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		boom.MethodNotAllowed(w, "Invalid request method")
		return
	}

	var body *HttpRequest
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	// Попытка декодировать JSON
	if err := decoder.Decode(&body); err != nil {
		boom.BadRequest(w, "Error parsing JSON")
		return
	}

	// Проверка тела на обязательные параметры
	if err := validateProxifyBody(body); err != nil {
		boom.BadRequest(w, err.Error())
		return
	}

	respBytes, err := callRequest(body)
	if err != nil {
		boom.BadRequest(w, err.Error())
	}

	httpResponse := &HttpResponse{
		Request:  body,
		Response: respBytes,
	}

	// Установка заголовка Content-Type
	w.Header().Set("Content-Type", "application/json")

	// Кодирование JSON и отправка ответа
	if err := json.NewEncoder(w).Encode(httpResponse); err != nil {
		boom.BadRequest(w, "Error encoding JSON")
	}
}

func Run() {
	// Initialize config
	_, err := config.NewConfig()
	if err != nil {
		logger.Errorf("[ENV] %s", err.Error())

		return
	}

	http.HandleFunc("/proxify", proxifyHandler)

	serverAddr := ":8080"
	logger.Errorf("Starting server on %s...\n", serverAddr)
	if err := http.ListenAndServe(serverAddr, nil); err != nil {
		logger.Errorf("Error starting server: %v\n", err)
	}
}
