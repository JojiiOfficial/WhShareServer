package main

const (
	//ServerError error from server
	ServerError string = "Server Error"
	//WrongInputFormatError wrong user input
	WrongInputFormatError string = "Wrong inputFormat!"
	//InvalidTokenError token is not valid
	InvalidTokenError string = "Token not valid"
	//BatchSizeTooLarge batch is too large
	BatchSizeTooLarge string = "BatchSize soo large!"
	//WrongIntegerFormat integer is probably no integer
	WrongIntegerFormat string = "Number is string"
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
	//HeaderStatus headername for status in response
	HeaderStatus string = "rstatus"
	//HeaderStatusMessage headername for status in response
	HeaderStatusMessage string = "rmess"
)
