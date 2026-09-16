package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpchelp "google.golang.org/grpc/status"
)

// ProbeRecord represents one sanitized ext_authz check event.
type ProbeRecord struct {
	Timestamp         string                 `json:"timestamp"`
	Method            string                 `json:"method"`
	Path              string                 `json:"path"`
	Headers           map[string]string      `json:"headers"`
	BodyPresent       bool                   `json:"body_present"`
	BodyLength        int64                  `json:"body_length"`
	RawBody           string                 `json:"raw_body"`
	Metadata          map[string]any         `json:"metadata"`
	ContextExtensions map[string]string      `json:"context_extensions"`
	ModeAtCheck       string                 `json:"mode_at_check"`
	OutcomeStatus     int32                  `json:"outcome_status"`
}

// ProbeServer implements Envoy's v3 ext_authz service.
type ProbeServer struct {
	authv3.UnimplementedAuthorizationServer
	mu      sync.RWMutex
	mode    string
	records []ProbeRecord
}

func NewProbeServer(initialMode string) *ProbeServer {
	if initialMode == "" {
		initialMode = "allow"
	}
	return &ProbeServer{
		mode:    initialMode,
		records: make([]ProbeRecord, 0),
	}
}

func (s *ProbeServer) SetMode(m string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = m
}

func (s *ProbeServer) GetMode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mode
}

func (s *ProbeServer) GetRecords() []ProbeRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copied := make([]ProbeRecord, len(s.records))
	copy(copied, s.records)
	return copied
}

func (s *ProbeServer) ResetRecords() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = make([]ProbeRecord, 0)
}

func sanitizeHeaderValue(key, val string) string {
	low := strings.ToLower(key)
	if low == "authorization" || low == "proxy-authorization" || strings.Contains(low, "token") || strings.Contains(low, "secret") {
		h := sha256.Sum256([]byte(val))
		return "sha256:" + hex.EncodeToString(h[:])
	}
	return val
}

func (s *ProbeServer) Check(ctx context.Context, req *authv3.CheckRequest) (*authv3.CheckResponse, error) {
	s.mu.Lock()
	currentMode := s.mode
	s.mu.Unlock()

	rec := ProbeRecord{
		Timestamp:         time.Now().UTC().Format(time.RFC3339Nano),
		Headers:           make(map[string]string),
		Metadata:          make(map[string]any),
		ContextExtensions: make(map[string]string),
		ModeAtCheck:       currentMode,
	}

	if req != nil && req.Attributes != nil {
		if req.Attributes.Request != nil && req.Attributes.Request.Http != nil {
			httpReq := req.Attributes.Request.Http
			rec.Method = httpReq.Method
			rec.Path = httpReq.Path
			for k, v := range httpReq.Headers {
				rec.Headers[k] = sanitizeHeaderValue(k, v)
			}

			// Capture body facts
			if len(httpReq.RawBody) > 0 {
				rec.BodyPresent = true
				rec.BodyLength = int64(len(httpReq.RawBody))
				rec.RawBody = string(httpReq.RawBody)
			} else if len(httpReq.Body) > 0 {
				rec.BodyPresent = true
				rec.BodyLength = int64(len(httpReq.Body))
				rec.RawBody = httpReq.Body
			}
		}

		if req.Attributes.ContextExtensions != nil {
			for k, v := range req.Attributes.ContextExtensions {
				rec.ContextExtensions[k] = v
			}
		}

		if req.Attributes.MetadataContext != nil && req.Attributes.MetadataContext.FilterMetadata != nil {
			for k, v := range req.Attributes.MetadataContext.FilterMetadata {
				rec.Metadata[k] = v.AsMap()
			}
		}
	}

	var resp *authv3.CheckResponse
	var err error

	switch currentMode {
	case "allow":
		rec.OutcomeStatus = int32(codes.OK)
		resp = &authv3.CheckResponse{
			Status: &status.Status{
				Code:    int32(codes.OK),
				Message: "allowed by g6 probe",
			},
			HttpResponse: &authv3.CheckResponse_OkResponse{
				OkResponse: &authv3.OkHttpResponse{
					Headers: []*corev3.HeaderValueOption{
						{
							Header: &corev3.HeaderValue{
								Key:   "x-agentgate-probe",
								Value: "allowed",
							},
						},
					},
				},
			},
		}

	case "deny":
		rec.OutcomeStatus = int32(codes.PermissionDenied)
		resp = &authv3.CheckResponse{
			Status: &status.Status{
				Code:    int32(codes.PermissionDenied),
				Message: "denied by g6 probe",
			},
			HttpResponse: &authv3.CheckResponse_DeniedResponse{
				DeniedResponse: &authv3.DeniedHttpResponse{
					Status: &typev3.HttpStatus{
						Code: typev3.StatusCode_Forbidden,
					},
					Body: "denied by g6 probe",
				},
			},
		}

	case "malformed":
		rec.OutcomeStatus = int32(codes.InvalidArgument)
		// Return invalid response without OkResponse or return non-OK error
		resp = &authv3.CheckResponse{
			Status: &status.Status{
				Code:    int32(codes.InvalidArgument),
				Message: "malformed request rejected by g6 probe",
			},
		}

	case "unavailable":
		rec.OutcomeStatus = int32(codes.Unavailable)
		err = grpchelp.Error(codes.Unavailable, "g6 probe authz service unavailable")

	default:
		rec.OutcomeStatus = int32(codes.Internal)
		err = errors.New("unknown authz mode: " + currentMode)
	}

	s.mu.Lock()
	s.records = append(s.records, rec)
	s.mu.Unlock()

	return resp, err
}

func startHTTPServer(addr string, probe *ProbeServer) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	mux.HandleFunc("/records", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		records := probe.GetRecords()
		_ = json.NewEncoder(w).Encode(records)
	})

	mux.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		probe.ResetRecords()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("reset"))
	})

	mux.HandleFunc("/mode", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			newMode := r.URL.Query().Get("mode")
			if newMode != "" {
				probe.SetMode(newMode)
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintf(w, "mode set to %s", newMode)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "current mode: %s", probe.GetMode())
	})

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return srv
}

func main() {
	grpcPort := os.Getenv("G6_AUTHZ_GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9001"
	}
	httpPort := os.Getenv("G6_AUTHZ_HTTP_PORT")
	if httpPort == "" {
		httpPort = "9002"
	}
	mode := os.Getenv("G6_AUTHZ_MODE")
	if mode == "" {
		mode = "allow"
	}

	probe := NewProbeServer(mode)

	// Start HTTP inspection server
	httpAddr := ":" + httpPort
	httpSrv := startHTTPServer(httpAddr, probe)
	log.Printf("Probe HTTP management server listening on %s (initial mode: %s)", httpAddr, mode)

	// Start gRPC ext_authz server
	grpcAddr := ":" + grpcPort
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen on gRPC %s: %v", grpcAddr, err)
	}
	grpcServer := grpc.NewServer()
	authv3.RegisterAuthorizationServer(grpcServer, probe)

	log.Printf("Probe gRPC ext_authz server listening on %s", grpcAddr)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down probe server...")
	grpcServer.GracefulStop()
	_ = httpSrv.Shutdown(context.Background())
	log.Println("Probe server exited cleanly.")
}
