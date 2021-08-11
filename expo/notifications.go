package expo

// Adapted from https://github.com/oliveroneill/exponent-server-sdk-golang

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
)

type PushMessage struct {
	To    string `json:"to"`
	Title string `json:"title,omitempty"`
	Body  string `json:"body"`
}

type Response struct {
	Data   []PushResponse      `json:"data"`
	Errors []map[string]string `json:"errors"`
}

type PushResponse struct {
	PushMessage PushMessage
	Status      string            `json:"status"`
	Message     string            `json:"message"`
	Details     map[string]string `json:"details"`
}

func (r *PushResponse) isSuccess() bool {
	return r.Status == SuccessStatus
}

func (r *PushResponse) ValidateResponse() error {
	if r.isSuccess() {
		return nil
	}
	err := &PushResponseError{
		Response: r,
	}
	// Handle specific errors if we have information
	if r.Details != nil {
		e := r.Details["error"]
		if e == ErrorDeviceNotRegistered {
			return &DeviceNotRegisteredError{
				PushResponseError: *err,
			}
		} else if e == ErrorMessageTooBig {
			return &MessageTooBigError{
				PushResponseError: *err,
			}
		} else if e == ErrorMessageRateExceeded {
			return &MessageRateExceededError{
				PushResponseError: *err,
			}
		}
	}
	return err
}

// SuccessStatus is the status returned from Expo on a success
const SuccessStatus = "ok"

// ErrorDeviceNotRegistered indicates the token is invalid
const ErrorDeviceNotRegistered = "DeviceNotRegistered"

// ErrorMessageTooBig indicates the message went over payload size of 4096 bytes
const ErrorMessageTooBig = "MessageTooBig"

// ErrorMessageRateExceeded indicates messages have been sent too frequently
const ErrorMessageRateExceeded = "MessageRateExceeded"

type PushResponseError struct {
	Response *PushResponse
}

func (e *PushResponseError) Error() string {
	if e.Response != nil {
		return e.Response.Message
	}
	return "Unknown push response error"
}

// DeviceNotRegisteredError is raised when the push token is invalid
// To handle this error, you should stop sending messages to this token.
type DeviceNotRegisteredError struct {
	PushResponseError
}

// MessageTooBigError is raised when the notification was too large.
// On Android and iOS, the total payload must be at most 4096 bytes.
type MessageTooBigError struct {
	PushResponseError
}

// MessageRateExceededError is raised when you are sending messages too frequently to a device
// You should implement exponential backoff and slowly retry sending messages.
type MessageRateExceededError struct {
	PushResponseError
}

type PushServerError struct {
	Message      string
	Response     *http.Response
	ResponseData *Response
	Errors       []map[string]string
}

// NewPushServerError creates a new PushServerError object
func NewPushServerError(message string, response *http.Response,
	responseData *Response,
	errors []map[string]string) *PushServerError {
	return &PushServerError{
		Message:      message,
		Response:     response,
		ResponseData: responseData,
		Errors:       errors,
	}
}

func (e *PushServerError) Error() string {
	return e.Message
}

func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return nil
	}
	return fmt.Errorf("invalid response (%d %s)", resp.StatusCode, resp.Status)
}

func PublishMessages(messages []PushMessage) ([]PushResponse, error) {
	path := "https://exp.host/--/api/v2/push/send"
	token := os.Getenv("EXPO_PUSH_ACCESS_TOKEN")
	if token == "" {
		return nil, errors.New("missing EXPO_PUSH_ACCESS_TOKEN")
	}

	var reqBody bytes.Buffer
	err := json.NewEncoder(&reqBody).Encode(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to json encode messages: %w", err)
	}

	req, err := http.NewRequest("POST", path, &reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create post request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post request failed: %w", err)
	}

	defer resp.Body.Close()

	// Check that we didn't receive an invalid response
	err = checkStatus(resp)
	if err != nil {
		return nil, err
	}

	// Validate the response format first
	var r *Response
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return nil, fmt.Errorf("failed to decode reponse json: %w", err)
	}
	// If there are errors with the entire request, return an error now.
	if r.Errors != nil {
		return nil, NewPushServerError("invalid server response", resp, r, r.Errors)
	}
	// We expect the response to have a 'data' field with the responses.
	if r.Data == nil {
		return nil, NewPushServerError("invalid server response", resp, r, nil)
	}
	// Sanity check the response
	if len(messages) != len(r.Data) {
		message := "Mismatched response length. Expected %d receipts but only received %d"
		errorMessage := fmt.Sprintf(message, len(messages), len(r.Data))
		return nil, NewPushServerError(errorMessage, resp, r, nil)
	}
	// Add the original message to each response for reference
	for i := range r.Data {
		r.Data[i].PushMessage = messages[i]
	}
	return r.Data, nil
}
