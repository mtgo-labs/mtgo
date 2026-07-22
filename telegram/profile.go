package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// SetProfilePhoto uploads and sets a new profile photo for the current user.
// The photo parameter must be a valid InputFileClass (e.g. an uploaded file).
// Returns an error if the client is not connected or the upload fails.
func (c *Client) SetProfilePhoto(ctx context.Context, photo tg.InputFileClass) error {
	if err := c.ensureConnectedContext(ctx); err != nil {
		return err
	}

	c.Log.Debug("SetProfilePhoto")

	rpc := c.Raw()
	_, err := rpc.PhotosUploadProfilePhoto(ctx, &tg.PhotosUploadProfilePhotoRequest{
		Flags: (1 << 0),
		File:  photo,
	})
	if err != nil {
		return fmt.Errorf("upload profile photo: %w", err)
	}

	return nil
}

// SetUsername updates the current user's username. The username must comply
// with Telegram's naming rules. Returns an error if the client is not
// connected or the update request fails.
func (c *Client) SetUsername(ctx context.Context, username string) error {
	if err := c.ensureConnectedContext(ctx); err != nil {
		return err
	}

	c.Log.Debugf("SetUsername username=%s", username)

	rpc := c.Raw()
	_, err := rpc.AccountUpdateUsername(ctx, &tg.AccountUpdateUsernameRequest{
		Username: username,
	})
	if err != nil {
		return fmt.Errorf("update username: %w", err)
	}

	return nil
}

// SetBio updates the current user's bio/about text. The bio must not exceed
// Telegram's length limit. Returns an error if the client is not connected
// or the update request fails.
func (c *Client) SetBio(ctx context.Context, bio string) error {
	if err := c.ensureConnectedContext(ctx); err != nil {
		return err
	}

	c.Log.Debug("SetBio")

	rpc := c.Raw()
	_, err := rpc.AccountUpdateProfile(ctx, &tg.AccountUpdateProfileRequest{
		About: bio,
	})
	if err != nil {
		return fmt.Errorf("update bio: %w", err)
	}

	return nil
}

// DeleteProfilePhoto deletes a profile photo by its ID. Returns an error if
// the client is not connected or the deletion request fails.
func (c *Client) DeleteProfilePhoto(ctx context.Context, photoID int64) error {
	if err := c.ensureConnectedContext(ctx); err != nil {
		return err
	}

	c.Log.Debugf("DeleteProfilePhoto id=%d", photoID)

	rpc := c.Raw()
	_, err := rpc.PhotosDeletePhotos(ctx, &tg.PhotosDeletePhotosRequest{
		ID: []tg.InputPhotoClass{&tg.InputPhoto{ID: photoID}},
	})
	if err != nil {
		return fmt.Errorf("delete profile photo: %w", err)
	}

	return nil
}

// GetProfilePhotosOption configures pagination and filtering options for
// GetProfilePhotos. All fields are optional; zero values are replaced with
// sensible defaults.
type GetProfilePhotosOption struct {
	// Offset is the number of photos to skip before returning results.
	Offset int32
	// Limit is the maximum number of photos to return. Defaults to 100 when
	// unset or zero.
	Limit int32
	// MaxID is the maximum photo ID to return; only photos with a smaller ID
	// are included. Use 0 to disable this filter.
	MaxID int64
}

// GetProfilePhotos retrieves a paginated list of profile photos for the
// specified user. Use GetProfilePhotosOption to control pagination.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - userID: identifier of the target user (UserRef)
//   - opts: optional GetProfilePhotosOption for pagination; defaults to
//     limit=100 when no options are provided
//
// Returns a slice of types.ChatPhoto representing the user's profile photos,
// or an error if the client is not connected, the user cannot be resolved, or
// the server returns an unexpected response type.
//
// Example:
//
//	photos, err := client.GetProfilePhotos(ctx, userID, &telegram.GetProfilePhotosOption{
//	    Limit: 10,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("User has %d profile photos\n", len(photos))
func (c *Client) GetProfilePhotos(ctx context.Context, userID int64, opts ...*GetProfilePhotosOption) ([]*types.ChatPhoto, error) {
	if err := c.ensureConnectedContext(ctx); err != nil {
		return nil, err
	}

	c.Log.Debugf("GetProfilePhotos user_id=%d", userID)

	opt := getOptDef(&GetProfilePhotosOption{Limit: 100}, opts...)
	if opt.Limit == 0 {
		opt.Limit = 100
	}

	user, err := resolveUserID(c, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve user: %w", err)
	}

	rpc := c.Raw()
	result, err := rpc.PhotosGetUserPhotos(ctx, &tg.PhotosGetUserPhotosRequest{
		UserID: user,
		Offset: opt.Offset,
		MaxID:  opt.MaxID,
		Limit:  opt.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("get user photos: %w", err)
	}

	var photos []tg.PhotoClass
	switch v := result.(type) {
	case *tg.PhotosPhotos:
		photos = v.Photos
	case *tg.PhotosPhotosSlice:
		photos = v.Photos
	default:
		return nil, fmt.Errorf("unexpected photos type %T", result)
	}

	out := make([]*types.ChatPhoto, 0, len(photos))
	for _, p := range photos {
		switch photo := p.(type) {
		case *tg.Photo:
			out = append(out, &types.ChatPhoto{
				SmallFileID: fmt.Sprintf("%d", photo.ID),
				BigFileID:   fmt.Sprintf("%d", photo.ID),
				DcID:        photo.DCID,
			})
		case *tg.PhotoEmpty:
		}
	}

	return out, nil
}
