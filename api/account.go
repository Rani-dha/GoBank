package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	db "GoBank/db/sqlc"
	"GoBank/token"
)

type createAccountRequest struct {
	Currency string `json:"currency" binding:"required,currency"`
}

// createAccount opens a new account for the authenticated user, one per currency.
// @Summary Open an account
// @Tags accounts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body createAccountRequest true "Currency (USD, EUR, or CAD)"
// @Success 200 {object} db.Account
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string "an account in this currency already exists for this owner"
// @Failure 500 {object} map[string]string
// @Router /accounts [post]
func (server *Server) createAccount(ctx *gin.Context) {
	var req createAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	arg := db.CreateAccountParams{
		Owner:    authPayload.Username,
		Currency: req.Currency,
		Balance:  0,
	}

	account, err := server.store.CreateAccount(ctx, arg)
	if err != nil {
		errCode := db.ErrorCode(err)
		if errCode == db.ForeignKeyViolation || errCode == db.UniqueViolation {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, account)
}

type getAccountRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// getAccount fetches one account by id. The account must belong to the caller.
// @Summary Get an account by id
// @Tags accounts
// @Produce json
// @Security BearerAuth
// @Param id path int true "Account ID"
// @Success 200 {object} db.Account
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string "account doesn't belong to the authenticated user"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /accounts/{id} [get]
func (server *Server) getAccount(ctx *gin.Context) {
	var req getAccountRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	account, err := server.store.GetAccount(ctx, req.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if account.Owner != authPayload.Username {
		err := errors.New("account doesn't belong to the authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, account)
}

type listAccountRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

// listAccounts pages through the authenticated user's own accounts.
// @Summary List your accounts
// @Tags accounts
// @Produce json
// @Security BearerAuth
// @Param page_id query int true "Page number, starting at 1"
// @Param page_size query int true "Accounts per page, 5-10"
// @Success 200 {array} db.Account
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /accounts [get]
func (server *Server) listAccounts(ctx *gin.Context) {
	var req listAccountRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	arg := db.ListAccountsParams{
		Owner:  authPayload.Username,
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	accounts, err := server.store.ListAccounts(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, accounts)
}

type lookupAccountRequest struct {
	Owner    string `form:"owner" binding:"required"`
	Currency string `form:"currency" binding:"required,currency"`
}

type lookupAccountResponse struct {
	ID int64 `json:"id"`
}

// lookupAccount resolves a (username, currency) pair to an account id, so a
// sender can address a transfer by who they're sending to instead of
// needing to already know a raw account id. Only the id is returned, not
// the account's balance or other details.
// @Summary Look up an account id by owner + currency
// @Tags accounts
// @Produce json
// @Security BearerAuth
// @Param owner query string true "Username of the account owner"
// @Param currency query string true "Currency (USD, EUR, or CAD)"
// @Success 200 {object} lookupAccountResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string "no account for that owner+currency"
// @Failure 500 {object} map[string]string
// @Router /accounts/lookup [get]
func (server *Server) lookupAccount(ctx *gin.Context) {
	var req lookupAccountRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	account, err := server.store.GetAccountByOwnerAndCurrency(ctx, db.GetAccountByOwnerAndCurrencyParams{
		Owner:    req.Owner,
		Currency: req.Currency,
	})
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, lookupAccountResponse{ID: account.ID})
}
