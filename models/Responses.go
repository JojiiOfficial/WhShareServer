package models

const (
	//NotFoundError error from server
	NotFoundError string = "Not found"
	//ActionNotAllowed error from server
	ActionNotAllowed string = "Action not allowed"
	//WrongLength error from server
	WrongLength string = "Wrong length"
	//ServerError error from server
	ServerError string = "Server Error"
	//WrongInputFormatError wrong user input
	WrongInputFormatError string = "Wrong inputFormat!"
	//InvalidTokenError token is not valid
	InvalidTokenError string = "Token not valid"
	//InvalidCallbackURL token is not valid
	InvalidCallbackURL string = "Callback url is invalid"
	//BatchSizeTooLarge batch is too large
	BatchSizeTooLarge string = "BatchSize soo large!"
	//WrongIntegerFormat integer is probably no integer
	WrongIntegerFormat string = "Number is string"
	//MultipleSourceNameErr err name already exists
	MultipleSourceNameErr string = "You can't have multiple sources with the same name"
	//UserIsInvalidErr err if user is invalid
	UserIsInvalidErr string = "user is invalid"
)

//ResponseStatus the status of response
type ResponseStatus uint8

const (
	//ResponseError if there was an error
	ResponseError ResponseStatus = 0
	//ResponseSuccess if the response is successful
	ResponseSuccess ResponseStatus = 1
)

const (
	//HeaderStatus headerName for status in response
	HeaderStatus string = "rstatus"
	//HeaderStatusMessage headerName for status in response
	HeaderStatusMessage string = "rmess"
)

//LoginResponse response for login
type LoginResponse struct {
	Token string `json:"token"`
}

//SourceAddResponse response for adding sources
type SourceAddResponse struct {
	Secret   string `json:"secret"`
	SourceID string `json:"id"`
}

//SubscriptionResponse response for subscription
type SubscriptionResponse struct {
	Message        string `json:"message,omitempty"`
	SubscriptionID string `json:"sid"`
	Name           string `json:"name"`
	Mode           uint8  `json:"mode"`
}

//ListSourcesResponse response containing a list of sources
type ListSourcesResponse struct {
	Sources []Source `json:"sources,omitempty"`
}
