package handlers

// Response represents common response
// @Description Common response
// @ID Response
type Response struct {
	// Reponse status (required field)
	Status string `json:"status"`

	// Reponse optional error message (optional field)
	Error string `json:"error,omitempty"`
}

const (
	StatusOK    = "OK"
	StatusError = "Error"
)

func RespError(msg string) Response {
	return Response{
		Status: StatusError,
		Error:  msg,
	}
}

func RespOK() Response {
	return Response{
		Status: StatusOK,
	}
}
