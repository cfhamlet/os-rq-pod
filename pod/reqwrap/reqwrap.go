package reqwrap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/cfhamlet/os-go-netloc-rule/matcher"
	"github.com/cfhamlet/os-go-netloc-rule/netloc"
	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/request"
	"github.com/cfhamlet/os-rq-pod/pkg/serv"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/cfhamlet/os-rq-pod/pod/global"
	"github.com/go-redis/redis/v7"
)

// SpcKeyReplace TODO
const SpcKeyReplace = "_replace_"

// AdminURI TODO
const AdminURI = "_ADMIN_"

// AdminNetloc TODO
var AdminNetloc = netloc.New(AdminURI, "", "")

// DefaultRequestConfig TODO
var DefaultRequestConfig = &RequestConfig{
	Meta: map[string]interface{}{
		"batch_id": "__UNKNOW__",
	},
}

// RequestConfig TODO
type RequestConfig struct {
	Method  string                 `json:"method,omitempty"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
	Headers map[string]string      `json:"headers,omitempty"`
	Cookies map[string]string      `json:"cookies,omitempty"`
	Body    string                 `json:"body,omitempty"`
}

// RequestWrapper TODO
type RequestWrapper struct {
	*serv.Serv
	client  *redis.Client
	matcher *matcher.Matcher
	*sync.RWMutex
}

// New TODO
func New(serv *serv.Serv, client *redis.Client) *RequestWrapper {
	return &RequestWrapper{serv, client, matcher.New(), &sync.RWMutex{}}
}

// Add TODO
func (wrapper *RequestWrapper) Add(nlc *netloc.Netloc, reqConfig *RequestConfig) (*netloc.Netloc, *RequestConfig, error) {
	wrapper.Lock()
	defer wrapper.Unlock()
	j, err := json.Marshal(reqConfig)
	if err == nil {
		_, err = wrapper.client.HSet(global.RedisRequestConfigKey, nlc.String(), string(j)).Result()
	}
	if err == nil {
		n, r := wrapper.matcher.Load(nlc, reqConfig)
		if r != nil {
			return n, r.(*RequestConfig), err
		}
	}
	return nil, nil, err
}

// Get TODO
func (wrapper *RequestWrapper) Get(nlc *netloc.Netloc) (*netloc.Netloc, *RequestConfig) {
	n, r := wrapper.matcher.Get(nlc)
	if r != nil {
		return n, r.(*RequestConfig)
	}
	return nil, nil
}

// NetlocRequestConfig TODO
type NetlocRequestConfig struct {
	Netloc *netloc.Netloc `json:"netloc"`
	Rule   *RequestConfig `json:"rule"`
}

// GetAll TODO
func (wrapper *RequestWrapper) GetAll() []*NetlocRequestConfig {
	out := []*NetlocRequestConfig{}
	wrapper.matcher.Iter(
		func(nlc *netloc.Netloc, rule interface{}) bool {
			out = append(out, &NetlocRequestConfig{nlc, rule.(*RequestConfig)})
			return true
		},
	)
	return out
}

// MatchURI TODO
func (wrapper *RequestWrapper) MatchURI(uri string) (*netloc.Netloc, *RequestConfig) {
	n, r := wrapper.matcher.MatchURL(uri)
	if r != nil {
		return n, r.(*RequestConfig)
	}
	return nil, nil
}

// Delete TODO
func (wrapper *RequestWrapper) Delete(nlc *netloc.Netloc) (*RequestConfig, error) {
	wrapper.Lock()
	defer wrapper.Unlock()
	n, _ := wrapper.matcher.Get(nlc)
	if n == nil {
		return nil, global.NotExistError(nlc.String())
	}
	_, err := wrapper.client.HDel(global.RedisRequestConfigKey, nlc.String()).Result()
	if err != nil {
		return nil, err
	}
	n, r := wrapper.matcher.Delete(nlc)
	if n != nil {
		return r.(*RequestConfig), nil
	}
	return nil, global.NotExistError(nlc.String())
}

// OnStart TODO
func (wrapper *RequestWrapper) OnStart(context.Context) error {
	err := wrapper.Load()
	if err == nil {
		nlcAdmin, _ := wrapper.matcher.Get(AdminNetloc)
		if nlcAdmin == nil {
			log.Logger.Warning("no _ADMIN_ request config")
			wrapper.loadAdminFromLocal()
		}
	}
	return err
}

// OnStop TODO
func (wrapper *RequestWrapper) OnStop(context.Context) error {
	return nil
}

func (wrapper *RequestWrapper) loadAdminFromLocal() {
	configAdmin := DefaultRequestConfig
	localConfig := wrapper.Conf().GetStringMap("request")
	if len(localConfig) > 0 {
		j, err := json.Marshal(localConfig)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(j, configAdmin)
		if err != nil {
			panic(err)
		}
	}
	wrapper.matcher.Load(netloc.New(AdminURI, "", ""), configAdmin)
}

// NetlocFromString TODO
func NetlocFromString(s string) (*netloc.Netloc, error) {
	c := strings.Split(s, "|")
	if len(c) != 3 {
		return nil, fmt.Errorf("'%s'", s)
	}
	return netloc.New(c[0], c[1], c[2]), nil
}

// Load TODO
func (wrapper *RequestWrapper) Load() error {
	scanner := utils.NewScanner(wrapper.client,
		"hscan", global.RedisRequestConfigKey, "*", 1000)

	log.Logger.Info("load request config start")
	err := scanner.Scan(
		func(keys []string) (err error) {
			isKey := false
			var nlc *netloc.Netloc
			for _, key := range keys {
				err = wrapper.SetStatus(serv.Preparing)
				if err != nil {
					break
				}
				isKey = !isKey
				if isKey {
					n, e := NetlocFromString(key)
					if e != nil {
						log.Logger.Warning(e)
					} else {
						nlc = n
					}
					continue
				}
				if nlc != nil {
					reqConf := &RequestConfig{}
					e := json.Unmarshal([]byte(key), &reqConf)
					if e != nil {
						log.Logger.Warning(nlc, e)
						continue
					}
					wrapper.matcher.Load(nlc, reqConf)
				}
			}
			return
		},
	)
	logf := log.Logger.Infof
	args := []interface{}{wrapper.matcher.Size(), "finish"}
	if err != nil {
		logf = log.Logger.Errorf
		args[1] = err
	}

	logf("load request config %d %v", args...)
	return err
}

// Cleanup TODO
func (wrapper *RequestWrapper) Cleanup() error {
	return nil
}

// MergeFunc TODO
type MergeFunc func(*request.Request, *RequestConfig, bool)

// Merge TODO
func (wrapper *RequestWrapper) Merge(req *request.Request, reqConfig *RequestConfig, force bool) {
	for _, f := range []MergeFunc{
		wrapper.mergeMethod,
		wrapper.mergeMeta,
		wrapper.mergeHeaders,
		wrapper.mergeCookies,
		wrapper.mergeBody,
	} {
		f(req, reqConfig, force)
	}
}

func (wrapper *RequestWrapper) mergeMethod(req *request.Request, reqConfig *RequestConfig, force bool) {
	if (force && reqConfig.Method != "") ||
		(req.Method == "" && reqConfig.Method != "") {
		req.Method = reqConfig.Method
	}
}

func (wrapper *RequestWrapper) mergeMeta(req *request.Request, reqConfig *RequestConfig, force bool) {
	if len(req.Meta) == 0 {
		if len(reqConfig.Meta) != 0 {
			req.Meta = reqConfig.Meta
			return
		}
		return
	}
	_, replace := req.Meta[SpcKeyReplace]
	if replace {
		delete(req.Meta, SpcKeyReplace)
		if force && len(reqConfig.Meta) != 0 {
			for k, v := range reqConfig.Meta {
				req.Meta[k] = v
			}
		}
		return
	}
	for k, v := range reqConfig.Meta {
		if force {
			req.Meta[k] = v
		} else {
			_, ok := req.Meta[k]
			if !ok {
				req.Meta[k] = v
			}
		}
	}
}

func (wrapper *RequestWrapper) mergeHeaders(req *request.Request, reqConfig *RequestConfig, force bool) {
	if len(req.Headers) == 0 {
		if len(reqConfig.Headers) != 0 {
			req.Headers = reqConfig.Headers
			return
		}
		return
	}
	wrapper.simpleMerge(req.Headers, reqConfig.Headers, force)
}

func (wrapper *RequestWrapper) mergeCookies(req *request.Request, reqConfig *RequestConfig, force bool) {
	if len(req.Cookies) == 0 {
		if len(reqConfig.Cookies) != 0 {
			req.Cookies = reqConfig.Cookies
			return
		}
		return
	}
	wrapper.simpleMerge(req.Cookies, reqConfig.Cookies, force)
}

func (wrapper *RequestWrapper) simpleMerge(dest, src map[string]string, force bool) {
	_, replace := dest[SpcKeyReplace]
	if replace {
		delete(dest, SpcKeyReplace)
		if force && len(src) != 0 {
			for k, v := range src {
				dest[k] = v
			}
		}
		return
	}
	for k, v := range src {
		if force {
			dest[k] = v
		} else {
			_, ok := dest[k]
			if !ok {
				dest[k] = v
			}
		}
	}
}

func (wrapper *RequestWrapper) mergeBody(req *request.Request, reqConfig *RequestConfig, force bool) {
	if len(req.Body) == 0 && len(reqConfig.Body) != 0 {
		req.Body = reqConfig.Body
	}
}

// Wrap TODO
func (wrapper *RequestWrapper) Wrap(req *request.Request) (*netloc.Netloc, *RequestConfig) {
	nlc, rule := wrapper.matcher.Match(req.Host, req.Port, req.Parsed.Scheme)
	var reqConfig *RequestConfig
	if nlc != nil {
		reqConfig = rule.(*RequestConfig)
		wrapper.Merge(req, reqConfig, false)
	}
	nlcAdmin, ruleAdmin := wrapper.matcher.Get(AdminNetloc)
	if nlcAdmin != nil {
		reqConfigAdmin := ruleAdmin.(*RequestConfig)
		wrapper.Merge(req, reqConfigAdmin, true)
	}
	return nlc, reqConfig
}
