package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	sharederrors "github.com/flighttracker/pkg/errors"
	"github.com/flighttracker/pkg/logger"

	"github.com/flighttracker/services/bot-service/internal/api"
)

const correlationIDHeader = "X-Correlation-Id"

// restClient is a small shared JSON-over-HTTP helper embedded by the
// flight-service and subscription-service clients. It is not exported —
// callers depend on the FlightClient/SubscriptionClient interfaces, not on
// this transport detail (repository pattern: callers depend on the port,
// not the adapter).
type restClient struct {
	baseURL    string
	httpClient *http.Client
}

// do issues an HTTP request with an optional JSON body, decodes a JSON
// response into out (if non-nil and status is 2xx), and maps non-2xx
// responses to *sharederrors.AppError using the shared ErrorResponse shape.
func (r *restClient) do(ctx context.Context, method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return sharederrors.Internal("CLIENT_MARSHAL_ERROR", "failed to encode request body", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, r.baseURL+path, reqBody)
	if err != nil {
		return sharederrors.Internal("CLIENT_REQUEST_ERROR", "failed to build request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if id := logger.CorrelationID(ctx); id != "" {
		req.Header.Set(correlationIDHeader, id)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return sharederrors.Unavailable("CLIENT_UNAVAILABLE", fmt.Sprintf("%s is unreachable", r.baseURL), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return sharederrors.Internal("CLIENT_READ_ERROR", "failed to read response body", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody api.ErrorResponseDTO
		_ = json.Unmarshal(respBody, &errBody)
		if errBody.Code == "" {
			errBody.Code = "UPSTREAM_ERROR"
		}
		return &sharederrors.AppError{
			Code:       errBody.Code,
			Message:    errBody.Message,
			HTTPStatus: resp.StatusCode,
		}
	}

	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return sharederrors.Internal("CLIENT_DECODE_ERROR", "failed to decode response body", err)
		}
	}
	return nil
}
