package insight

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/dynamicgo/config"
	"github.com/dynamicgo/slf4go"
	"github.com/julienschmidt/httprouter"
	"github.com/ybbus/jsonrpc"
)

type handler func(params []interface{}) (interface{}, *JSONRPCError)

// Server insight api jsonrpc 2.0 server
type Server struct {
	slf4go.Logger
	conf          *config.Config
	router        *httprouter.Router
	remote        *url.URL
	dispatch      map[string]handler
	timeEstimator *BlockTimeEstimator
}

// NewServer create new server
func NewServer(cnf *config.Config) (*Server, error) {

	remote, err := url.Parse(cnf.GetString("insight.geth", "http://xxxxxx:10332"))

	if err != nil {
		return nil, err
	}

	return &Server{
		Logger:        slf4go.Get("geth-insight"),
		conf:          cnf,
		router:        httprouter.New(),
		remote:        remote,
		dispatch:      make(map[string]handler),
		timeEstimator: newBlockTimeEstimator(cnf),
	}, nil
}

type loggerHandler struct {
	slf4go.Logger
	handler http.Handler
}

func (l *loggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l.DebugF("http route: %s %s", r.Method, r.URL)
	l.handler.ServeHTTP(w, r)
}

// Run insight server
func (server *Server) Run() {

	server.router.POST("/", server.dispatchRequest)

	server.dispatch["blockPerSecond"] = server.blockPerSecond

	server.Fatal(http.ListenAndServe(
		server.conf.GetString("insight.listen", ":8545"),
		&loggerHandler{
			Logger:  server.Logger,
			handler: server.router,
		},
	))
}

func (server *Server) blockPerSecond(params []interface{}) (interface{}, *JSONRPCError) {
	return server.timeEstimator.getBPS(), nil
}

func (server *Server) decodeRPCRequest(r *http.Request) (*jsonrpc.RPCRequest, error) {
	request := jsonrpc.RPCRequest{}

	decoder := json.NewDecoder(r.Body)

	decoder.UseNumber()

	err := decoder.Decode(&request)

	if err != nil {
		return nil, err
	}

	return &request, nil
}

func (server *Server) makeJSONRPCError(w http.ResponseWriter, id uint, code int, message string, data interface{}) {
	response := &jsonrpc.RPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &jsonrpc.RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	jsonresponse, err := json.Marshal(response)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, "server internal error", http.StatusInternalServerError)
		server.ErrorF("marshal response error :%s", err)
		return
	}

	w.WriteHeader(200)

	if _, err := w.Write(jsonresponse); err != nil {
		server.ErrorF("write response error :%s", err)
	}
}

func (server *Server) makeJSONRPCResponse(w http.ResponseWriter, id uint, data interface{}) {
	response := &jsonrpc.RPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  data,
	}

	jsonresponse, err := json.Marshal(response)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, "server internal error", http.StatusInternalServerError)
		server.ErrorF("marshal response error :%s", err)
		return
	}

	w.WriteHeader(200)

	if _, err := w.Write(jsonresponse); err != nil {
		server.ErrorF("write response error :%s", err)
	}
}

func (server *Server) dispatchRequest(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	request, err := server.decodeRPCRequest(r)

	if err != nil {
		server.makeJSONRPCError(w, 0, JSONRPCParserError, "parse error", nil)
		return
	}

	if method, ok := server.dispatch[request.Method]; ok {

		if request.Params == nil {
			request.Params = make([]interface{}, 0)
		}

		result, err := method(request.Params.([]interface{}))

		if err != nil {
			server.makeJSONRPCError(w, request.ID, err.ID, err.Message, result)
		} else {
			server.makeJSONRPCResponse(w, request.ID, result)
		}

	} else {

		data, err := json.Marshal(request)

		if err != nil {
			server.makeJSONRPCError(w, request.ID, JSONRPCInnerError, err.Error(), nil)
			return
		}

		r.Body = ioutil.NopCloser(bytes.NewReader(data))

		reverseProxy := httputil.NewSingleHostReverseProxy(server.remote)

		reverseProxy.ServeHTTP(w, r)
	}
}
