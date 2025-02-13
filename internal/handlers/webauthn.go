package handlers

import (
	"fmt"
	"net/url"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/authelia/authelia/v4/internal/middlewares"
	"github.com/authelia/authelia/v4/internal/model"
	"github.com/authelia/authelia/v4/internal/session"
)

func getWebAuthnUser(ctx *middlewares.AutheliaCtx, userSession session.UserSession) (user *model.WebAuthnUser, err error) {
	user = &model.WebAuthnUser{
		Username:    userSession.Username,
		DisplayName: userSession.DisplayName,
	}

	if user.DisplayName == "" {
		user.DisplayName = user.Username
	}

	if user.Devices, err = ctx.Providers.StorageProvider.LoadWebAuthnDevicesByUsername(ctx, userSession.Username); err != nil {
		return nil, err
	}

	var (
		opaqueID *model.UserOpaqueIdentifier
	)

	if opaqueID, err = getWebAuthnUserOpaqueID(ctx, user.Username); err != nil {
		return nil, err
	}

	user.UserID = opaqueID.Identifier.String()

	return user, nil
}

func getWebAuthnUserOpaqueID(ctx *middlewares.AutheliaCtx, username string) (opaqueID *model.UserOpaqueIdentifier, err error) {
	if opaqueID, err = ctx.Providers.StorageProvider.LoadUserOpaqueIdentifierBySignature(ctx, "webauthn", "pre", username); err != nil {
		return nil, err
	} else if opaqueID == nil {
		if opaqueID, err = model.NewUserOpaqueIdentifier("webauthn", "pre", username); err != nil {
			return nil, err
		}

		if err = ctx.Providers.StorageProvider.SaveUserOpaqueIdentifier(ctx, *opaqueID); err != nil {
			return nil, err
		}
	}

	return opaqueID, nil
}

func newWebAuthn(ctx *middlewares.AutheliaCtx) (w *webauthn.WebAuthn, err error) {
	var (
		u *url.URL
	)

	if u, err = ctx.GetXOriginalURLOrXForwardedURL(); err != nil {
		return nil, err
	}

	rpID := u.Hostname()
	origin := fmt.Sprintf("%s://%s", u.Scheme, u.Host)

	config := &webauthn.Config{
		RPDisplayName: ctx.Configuration.WebAuthn.DisplayName,
		RPID:          rpID,
		RPOrigin:      origin,
		RPIcon:        "",

		AttestationPreference: ctx.Configuration.WebAuthn.ConveyancePreference,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.CrossPlatform,
			UserVerification:        ctx.Configuration.WebAuthn.UserVerification,
			RequireResidentKey:      protocol.ResidentKeyNotRequired(),
		},

		Timeout: int(ctx.Configuration.WebAuthn.Timeout.Milliseconds()),
	}

	ctx.Logger.Tracef("Creating new WebAuthn RP instance with ID %s and Origins %s", config.RPID, config.RPOrigin)

	return webauthn.New(config)
}
