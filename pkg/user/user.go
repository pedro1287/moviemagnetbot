package user

import (
	"fmt"
	"os"
	"time"

	"github.com/go-pg/pg/orm"
	"github.com/gorilla/feeds"
	"github.com/speps/go-hashids"

	"github.com/pedro1287/moviemagnetbot/pkg/db"
	"github.com/pedro1287/moviemagnetbot/pkg/torrent"
)

const (
	hashAlphabet = "f52b5a057b73b9974eaa7403e04907f0"

	userFeedTitle = "BotStart"
	userFeedURL   = "https://moviemagnetbot.herokuapp.com/tasks/%s.xml"

	itemsPerFeed       = 20
	feedCheckThreshold = 24 * time.Hour
)

// User (i.e. Downloader)
type User struct {
	ID            int
	CreatedAt     time.Time
	UpdatedAt     time.Time
	TelegramID    int
	TelegramName  string
	FeedID        string
	FeedCheckedAt time.Time
	Torrents      []torrent.Torrent `pg:",many2many:user_torrents"`
}

// UserTorrent is about which user download what torrents
type UserTorrent struct { // nolint
	UserID               int
	TorrentID            int
	Torrent_DownloadedAt time.Time // nolint
}

func (u *User) create() (*User, error) {
	_, err := db.DB.Model(u).
		Where("telegram_id= ?telegram_id").
		OnConflict("DO NOTHING").
		SelectOrInsert()
	if err != nil {
		return nil, err
	}
	return u.generateFeedID()
}

func (u *User) generateFeedID() (*User, error) {
	u, err := u.newFeedID()
	if err != nil {
		return nil, err
	}
	err = u.update()
	return u, err
}

func (u *User) newFeedID() (*User, error) {
	hd := hashids.NewData()
	hd.Salt = os.Getenv("MOVIE_MAGNET_BOT_SALT")
	hd.Alphabet = hashAlphabet
	h, err := hashids.NewWithData(hd)
	if err != nil {
		return nil, err
	}
	feed, err := h.Encode([]int{u.TelegramID})
	if err != nil {
		return nil, err
	}
	u.FeedID = feed
	return u, nil
}

func (u *User) getByTelegramID() (*User, error) {
	u, err := u.create()
	if err != nil {
		return nil, err
	}
	err = db.DB.Model(u).Where("telegram_id = ?", u.TelegramID).Select()
	return u, err
}

// GetByFeedID find User by FeedID
func (u *User) GetByFeedID() (*User, error) {
	err := db.DB.Model(u).Where("feed_id = ?", u.FeedID).Select()
	return u, err
}

// AppendTorrent apeend Torrent to User
func (u *User) AppendTorrent(t *torrent.Torrent) error {
	u, err := u.getByTelegramID()
	if err != nil {
		return err
	}
	return db.DB.Insert(&UserTorrent{u.ID, t.ID, time.Now()})
}

func (u *User) getTorrents(limit int) ([]torrent.Torrent, error) {
	err := db.DB.Model(u).Column("user.*", "Torrents").
		Relation("Torrents", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("torrent__downloaded_at DESC").Limit(limit), nil
		}).
		Where("id = ?", u.ID).Select()
	return u.Torrents, err
}

// CountTorrents count User Torrents
func (u *User) CountTorrents() (int, error) {
	u, err := u.getByTelegramID()
	if err != nil {
		return 0, err
	}
	res, err := db.DB.Model((*UserTorrent)(nil)).Where("user_id = ?", u.ID).Count()
	if err != nil {
		return 0, err
	}
	return res, nil
}

// ClearTorrents clear User Torrent history
func (u *User) ClearTorrents() (int, error) {
	u, err := u.getByTelegramID()
	if err != nil {
		return 0, err
	}
	res, err := db.DB.Model((*UserTorrent)(nil)).Where("user_id = ?", u.ID).Delete()
	if err != nil {
		return 0, err
	}
	return res.RowsAffected(), nil
}

func (u *User) update() error {
	return db.DB.Update(u)
}

// FeedURL returns User’s feed URL
func (u *User) FeedURL() string {
	return fmt.Sprintf(userFeedURL, u.FeedID)
}

// GenerateFeed returns User’s feed
func (u *User) GenerateFeed() (string, error) {
	feed := &feeds.Feed{
		Title:   userFeedTitle,
		Link:    &feeds.Link{Href: u.FeedURL()},
		Created: time.Now(),
	}
	torrents, err := u.getTorrents(itemsPerFeed)
	if err != nil {
		return "", err
	}
	for _, t := range torrents {
		if t.Title == "" {
			t.Title = t.Magnet
		}

		feed.Items = append(feed.Items, &feeds.Item{
			Title:   t.Title,
			Link:    &feeds.Link{Href: t.Magnet},
			Created: t.DownloadedAt,
		})
	}
	rss, err := feed.ToRss()
	if err != nil {
		return "", err
	}
	return rss, nil
}

// RenewFeedChecked renews User’s FeedCheckedAt
func (u *User) RenewFeedChecked() error {
	u.FeedCheckedAt = time.Now()
	return u.update()
}

// IsFeedActive tells if the User’s feed has been requested recently
func (u *User) IsFeedActive() bool {
	return time.Since(u.FeedCheckedAt) < feedCheckThreshold
}

// BeforeInsert hook
func (u *User) BeforeInsert(db orm.DB) error {
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}
	return nil
}

// BeforeUpdate hook
func (u *User) BeforeUpdate(db orm.DB) error {
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = time.Now()
	}
	return nil
}
