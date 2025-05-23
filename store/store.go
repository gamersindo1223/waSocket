// Copyright (c) 2022 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package store contains interfaces for storing data needed for WhatsApp multidevice.
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/techwiz37/waSocket/proto/waAdv"
	"github.com/techwiz37/waSocket/types"
	"github.com/techwiz37/waSocket/util/keys"
	waLog "github.com/techwiz37/waSocket/util/log"
)

type IdentityStore interface {
	PutIdentity(address string, key [32]byte) error
	DeleteAllIdentities(phone string) error
	DeleteIdentity(address string) error
	IsTrustedIdentity(address string, key [32]byte) (bool, error)
}

type SessionStore interface {
	GetSession(address string) ([]byte, error)
	HasSession(address string) (bool, error)
	PutSession(address string, session []byte) error
	DeleteAllSessions(phone string) error
	DeleteSession(address string) error
	MigratePNToLID(ctx context.Context, pn, lid types.JID) error
}

type PreKeyStore interface {
	GetOrGenPreKeys(count uint32) ([]*keys.PreKey, error)
	GenOnePreKey() (*keys.PreKey, error)
	GetPreKey(id uint32) (*keys.PreKey, error)
	RemovePreKey(id uint32) error
	MarkPreKeysAsUploaded(upToID uint32) error
	UploadedPreKeyCount() (int, error)
}

type SenderKeyStore interface {
	PutSenderKey(group, user string, session []byte) error
	GetSenderKey(group, user string) ([]byte, error)
}

type AppStateSyncKey struct {
	Data        []byte
	Fingerprint []byte
	Timestamp   int64
}

type AppStateSyncKeyStore interface {
	PutAppStateSyncKey(id []byte, key AppStateSyncKey) error
	GetAppStateSyncKey(id []byte) (*AppStateSyncKey, error)
	GetLatestAppStateSyncKeyID() ([]byte, error)
}

type AppStateMutationMAC struct {
	IndexMAC []byte
	ValueMAC []byte
}

type AppStateStore interface {
	PutAppStateVersion(name string, version uint64, hash [128]byte) error
	GetAppStateVersion(name string) (uint64, [128]byte, error)
	DeleteAppStateVersion(name string) error

	PutAppStateMutationMACs(name string, version uint64, mutations []AppStateMutationMAC) error
	DeleteAppStateMutationMACs(name string, indexMACs [][]byte) error
	GetAppStateMutationMAC(name string, indexMAC []byte) (valueMAC []byte, err error)
}

type ContactEntry struct {
	JID       types.JID
	FirstName string
	FullName  string
}

type ContactStore interface {
	PutPushName(user types.JID, pushName string) (bool, string, error)
	PutBusinessName(user types.JID, businessName string) (bool, string, error)
	PutContactName(user types.JID, fullName, firstName string) error
	PutAllContactNames(contacts []ContactEntry) error
	GetContact(user types.JID) (types.ContactInfo, error)
	GetAllContacts() (map[types.JID]types.ContactInfo, error)
}

type ChatSettingsStore interface {
	PutMutedUntil(chat types.JID, mutedUntil time.Time) error
	PutPinned(chat types.JID, pinned bool) error
	PutArchived(chat types.JID, archived bool) error
	GetChatSettings(chat types.JID) (types.LocalChatSettings, error)
}

type DeviceContainer interface {
	PutDevice(store *Device) error
	DeleteDevice(store *Device) error
}

type MessageSecretInsert struct {
	Chat   types.JID
	Sender types.JID
	ID     types.MessageID
	Secret []byte
}

type MsgSecretStore interface {
	PutMessageSecrets([]MessageSecretInsert) error
	PutMessageSecret(chat, sender types.JID, id types.MessageID, secret []byte) error
	GetMessageSecret(chat, sender types.JID, id types.MessageID) ([]byte, error)
}

type PrivacyToken struct {
	User      types.JID
	Token     []byte
	Timestamp time.Time
}

type PrivacyTokenStore interface {
	PutPrivacyTokens(tokens ...PrivacyToken) error
	GetPrivacyToken(user types.JID) (*PrivacyToken, error)
}

type LIDMapping struct {
	LID types.JID
	PN  types.JID
}

type LIDStore interface {
	PutManyLIDMappings(ctx context.Context, mappings []LIDMapping) error
	PutLIDMapping(ctx context.Context, lid, jid types.JID) error
	GetPNForLID(ctx context.Context, lid types.JID) (types.JID, error)
	GetLIDForPN(ctx context.Context, pn types.JID) (types.JID, error)
}

type AllSessionSpecificStores interface {
	IdentityStore
	SessionStore
	PreKeyStore
	SenderKeyStore
	AppStateSyncKeyStore
	AppStateStore
	ContactStore
	ChatSettingsStore
	MsgSecretStore
	PrivacyTokenStore
}

type AllGlobalStores interface {
	LIDStore
}

type AllStores interface {
	AllSessionSpecificStores
	AllGlobalStores
}

type Device struct {
	Log waLog.Logger

	NoiseKey       *keys.KeyPair
	IdentityKey    *keys.KeyPair
	SignedPreKey   *keys.PreKey
	RegistrationID uint32
	AdvSecretKey   []byte

	ID           *types.JID
	LID          types.JID
	Account      *waAdv.ADVSignedDeviceIdentity
	Platform     string
	BusinessName string
	PushName     string

	FacebookUUID uuid.UUID

	Initialized   bool
	Identities    IdentityStore
	Sessions      SessionStore
	PreKeys       PreKeyStore
	SenderKeys    SenderKeyStore
	AppStateKeys  AppStateSyncKeyStore
	AppState      AppStateStore
	Contacts      ContactStore
	ChatSettings  ChatSettingsStore
	MsgSecrets    MsgSecretStore
	PrivacyTokens PrivacyTokenStore
	LIDs          LIDStore
	Container     DeviceContainer

	DatabaseErrorHandler func(device *Device, action string, attemptIndex int, err error) (retry bool)
}

func (device *Device) handleDatabaseError(attemptIndex int, err error, action string, args ...interface{}) bool {
	if device.DatabaseErrorHandler != nil {
		return device.DatabaseErrorHandler(device, fmt.Sprintf(action, args...), attemptIndex, err)
	}
	device.Log.Errorf("Failed to %s: %v", fmt.Sprintf(action, args...), err)
	return false
}

func (device *Device) GetJID() types.JID {
	if device == nil {
		return types.EmptyJID
	}
	id := device.ID
	if id == nil {
		return types.EmptyJID
	}
	return *id
}

func (device *Device) GetLID() types.JID {
	if device == nil {
		return types.EmptyJID
	}
	return device.LID
}

func (device *Device) Save() error {
	return device.Container.PutDevice(device)
}

func (device *Device) Delete() error {
	err := device.Container.DeleteDevice(device)
	if err != nil {
		return err
	}
	device.ID = nil
	device.LID = types.EmptyJID
	return nil
}
