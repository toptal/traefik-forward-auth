package tfa

import (
	"io/ioutil"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
)

type Directory struct {
	cache   sync.Map
	service *admin.Service
}

type CacheEntry struct {
	Groups []string
	TTL    int64
}

func NewDirectory() *Directory {
	return &Directory{service: nil}
}

func (d *Directory) IsMember(email string, group string) bool {
	for _, g := range d.groups(email) {
		if g == group {
			return true
		}
	}
	return false
}

func (d *Directory) getCache(email string) *CacheEntry {
	if cacheEntry, ok := d.cache.Load(email); ok {
		cacheEntry := cacheEntry.(CacheEntry)
		return &cacheEntry
	}
	// Email is not found in the cache.
	return nil
}

func (d *Directory) setCache(email string, groups []string, ttl int64) {
	cacheEntry := CacheEntry{
		Groups: groups,
		TTL:    ttl,
	}
	d.cache.Store(email, cacheEntry)
}

func (d *Directory) deleteCache(email string) {
	d.cache.Delete(email)
}

func (d *Directory) groups(email string) []string {
	groups := []string{}

	cacheEntry := d.getCache(email)
	if cacheEntry == nil || time.Now().Unix() > cacheEntry.TTL {
		if list, err := d.getGroups(email); err == nil {
			groups = list
			log.WithFields(logrus.Fields{"email": email, "groups": groups}).Debug("Fetched groups from API")
			ttl := time.Now().Unix() + config.GoogleExpirySeconds
			d.setCache(email, list, ttl)
		} else {
			log.WithFields(logrus.Fields{"email": email}).Debug("Failed to fetch groups from API")
			log.Error(err)
			d.deleteCache(email)
		}
	} else {
		groups = cacheEntry.Groups
		log.WithFields(logrus.Fields{"email": email, "groups": groups}).Debug("Using groups from cache")
	}

	return groups
}

func (d *Directory) getGroups(email string) ([]string, error) {
	if d.service == nil {
		srv, err := d.createService()
		if err != nil {
			return nil, err
		}
		d.service = srv
	}

	groups, err := d.service.Groups.List().Domain(config.GoogleDomain).UserKey(email).MaxResults(200).Do()
	if err != nil {
		return nil, err
	}
	var list []string
	for _, g := range groups.Groups {
		list = append(list, g.Email)
	}
	return list, nil
}

func (d *Directory) createService() (*admin.Service, error) {
	json, err := ioutil.ReadFile(config.GoogleApplicationCredentials)
	if err != nil {
		return nil, err
	}
	jwt, err := google.JWTConfigFromJSON(json, admin.AdminDirectoryGroupReadonlyScope)
	if err != nil {
		return nil, err
	}
	jwt.Subject = config.GoogleActingAdminEmail
	ctx := context.Background()
	client := jwt.Client(ctx)
	srv, err := admin.New(client)
	if err != nil {
		return nil, err
	}
	return srv, nil
}
