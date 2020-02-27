package handlers

import (
	"errors"
	"net/http"
	"strings"

	dbhelper "github.com/JojiiOfficial/GoDBHelper"
	"github.com/JojiiOfficial/WhShareServer/models"
)

var (
	//ErrorTokenInvalid error if token is invalid
	ErrorTokenInvalid error = errors.New("Token invalid")
	//ErrorTokenEmpty error if token is empty
	ErrorTokenEmpty error = errors.New("Token empty")
)

//AuthHandler handler for http auth
type AuthHandler struct {
	Request *http.Request
	db      *dbhelper.DBhelper
}

//NewAuthHandler returns a new AuthHandler
func NewAuthHandler(request *http.Request, db *dbhelper.DBhelper) *AuthHandler {
	return &AuthHandler{
		Request: request,
		db:      db,
	}
}

//GetBearer return the bearer token
func (authHandler AuthHandler) GetBearer() string {
	authHeader, has := authHandler.Request.Header["Authorization"]
	//Validate bearer token
	if !has || len(authHeader) == 0 || !strings.HasPrefix(authHeader[0], "Bearer") || len(tokenFromBearerHeader(authHeader[0])) != 64 {
		return ""
	}
	return tokenFromBearerHeader(authHeader[0])
}

func tokenFromBearerHeader(header string) string {
	return strings.TrimSpace(strings.ReplaceAll(header, "Bearer", ""))
}

//GetUserFromBearer returns the User assigned to the token
func (authHandler AuthHandler) GetUserFromBearer() (*models.User, error) {
	token := authHandler.GetBearer()
	if len(token) == 0 {
		return nil, ErrorTokenEmpty
	}

	if len(token) > 0 && len(token) != 64 {
		return nil, ErrorTokenInvalid
	}

	return models.GetUserBySession(authHandler.db, token)
}

//IsInvalid return true if err is invalid
func (authHandler AuthHandler) IsInvalid(err error) bool {
	return err == ErrorTokenInvalid
}
