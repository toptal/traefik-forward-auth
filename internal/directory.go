package tfa

import (
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/admin/directory/v1"
	"io/ioutil"
	"time"
)

type Directory struct {
	cache                        map[string][]string
	ttl                          map[string]int64
	domain                       string
	googleApplicationCredentials string
	actingAdminEmail             string
	expirySeconds                int64
}

func NewDirectory() *Directory {
	cache := make(map[string][]string)
	ttl := make(map[string]int64)
	domain := config.GoogleDomain
	googleApplicationCredentials := config.GoogleApplicationCredentials
	actingAdminEmail := config.GoogleActingAdminEmail
	expirySeconds := config.GoogleExpirySeconds
	return &Directory{cache, ttl, domain, googleApplicationCredentials, actingAdminEmail, expirySeconds}
}

func (d *Directory) IsMember(email string, group string) bool {
	for _, g := range d.groups(email) {
		if g == group {
			return true
		}
	}
	return false
}

func (d *Directory) groups(email string) []string {
	if ttl, ok := d.ttl[email]; !ok || time.Now().Unix() > ttl {
		if list, err := d.getGroups(email); err == nil {
			log.WithFields(logrus.Fields{"email": email}).Info("Fetched groups from API")
			d.cache[email] = list
			d.ttl[email] = time.Now().Unix() + d.expirySeconds
		} else {
			log.Error(err)
			delete(d.cache, email)
			delete(d.ttl, email)
		}
	}

	if groups, ok := d.cache[email]; ok {
		return groups
	}
	return []string{}
}

func (d *Directory) getGroups(email string) ([]string, error) {
	srv, err := d.createService()
	if err != nil {
		return nil, err
	}

	groups, err := srv.Groups.List().Domain(d.domain).UserKey(email).MaxResults(200).Do()
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
	json, err := ioutil.ReadFile(d.googleApplicationCredentials)
	if err != nil {
		return nil, err
	}
	config, err := google.JWTConfigFromJSON(json, admin.AdminDirectoryGroupReadonlyScope)
	if err != nil {
		return nil, err
	}
	config.Subject = d.actingAdminEmail
	ctx := context.Background()
	client := config.Client(ctx)
	srv, err := admin.New(client)
	if err != nil {
		return nil, err
	}
	return srv, nil
}
