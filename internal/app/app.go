package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/darahayes/go-boom"
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
		Type    string            `json:"response_type"`
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

func callRequest(request *HttpRequest) (*http.Response, error) {
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

	if len(requestParams) > 0 {
		requestUrl.RawQuery = requestParams.Encode()
	}

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
	case "patch":
		requestMethod = http.MethodPatch
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

	return resp, nil
}

func validateProxifyBody(body *HttpRequest) error {
	if body.Url == "" {
		return errors.New("request.url is required")
	}
	if body.Method == "" {
		return errors.New("request.method is required")
	}

	// Проверка наличия Proxy и его обязательных полей
	if body.Proxy != nil {
		if body.Proxy.Host == "" {
			return errors.New("request.proxy.host is required")
		}
		if body.Proxy.ProxyType == "" {
			return errors.New("request.proxy.type is required")
		}
	}

	// Проверка допустимых типов ответа
	if body.Type != "binary" && body.Type != "json" && body.Type != "" {
		return fmt.Errorf("request.response_type = '%s' unsupported. Supported types: json, binary", body.Type)
	}

	return nil
}

func proxifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		boom.MethodNotAllowed(w, "Invalid request method")
		return
	}

	var body HttpRequest
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	// Попытка декодировать JSON
	if err := decoder.Decode(&body); err != nil {
		boom.BadData(w, "Error parsing JSON")
		return
	}

	// Проверка тела на обязательные параметры
	if err := validateProxifyBody(&body); err != nil {
		boom.BadRequest(w, err.Error())
		return
	}

	resp, err := callRequest(&body)
	if err != nil {
		boom.BadRequest(w, err.Error())
		return
	}
	defer resp.Body.Close() // Закрываем тело ответа

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		boom.BadData(w, "Error reading response body")
		return
	}

	if body.Type == "binary" {
		// Установка заголовка Content-Type
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))

		// Копируем данные ответа в тело ответа
		_, err = w.Write(respBytes)
		if err != nil {
			boom.BadData(w, "Unable to write binary")
		}
	} else {
		httpResponse := &HttpResponse{
			Request:  &body,
			Response: respBytes,
		}

		// Установка заголовка Content-Type
		w.Header().Set("Content-Type", "application/json")

		// Кодирование JSON и отправка ответа
		if err := json.NewEncoder(w).Encode(httpResponse); err != nil {
			boom.BadData(w, "Error encoding JSON")
		}
	}
}

func Run() {
	http.HandleFunc("/proxify", proxifyHandler)
	http.HandleFunc("/", (func(w http.ResponseWriter, r *http.Request) { boom.NotFound(w, "Page not Found!") }))

	serverAddr := ":8080"
	logger.Errorf("Starting server on %s...\n", serverAddr)
	if err := http.ListenAndServe(serverAddr, nil); err != nil {
		logger.Errorf("Error starting server: %v\n", err)
	}
}
