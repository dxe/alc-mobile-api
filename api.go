package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"

	"github.com/dxe/alc-mobile-api/model"
)

type api struct {
	// query is the underlying SQL query that the API issues to the
	// MySQL database. It should return a single-row, single-column JSON
	// result.
	query string

	// args returns a pointer to a newly allocated variable able to
	// store the arguments from the JSON request body.
	args func() interface{}

	// value returns a pointer to a newly allocated Go variable able to
	// represent the JSON object returned by the query.
	value func() interface{}
}

func (a *api) serve(s *server) {
	var queryArgs interface{}

	if a.args != nil {
		args := a.args()
		err := json.NewDecoder(s.r.Body).Decode(args)
		if err != nil {
			a.error(s, fmt.Errorf("failed to decode json request body: %w", err))
			return
		}
		queryArgs = args
	}

	if a.value == nil {
		if _, err := s.db.NamedExecContext(s.r.Context(), a.query, queryArgs); err != nil {
			a.error(s, err)
		}
		return
	}

	// TODO(mdempsky): Implement caching and/or single-flighting (e.g.,
	// golang.org/x/sync/singleflight), so we don't need to issue a DB
	// request for each HTTP request.

	var buf []byte
	var result *sqlx.Rows
	var err error

	if queryArgs == nil {
		result, err = s.db.QueryxContext(s.r.Context(), a.query)
	} else {
		result, err = s.db.NamedQueryContext(s.r.Context(), a.query, queryArgs)
	}

	if err != nil {
		a.error(s, err)
		return
	}
	result.Next()
	if err := result.Scan(&buf); err != nil {
		a.error(s, err)
		return
	}

	if !*flagProd {
		// When not in production, check that the SQL response decodes
		// correctly.
		// TODO(mdempsky): Replace this with unit tests.
		if err := json.NewDecoder(bytes.NewReader(buf)).Decode(a.value()); err != nil {
			log.Printf("failed to decode JSON response: %v", err)
		}
	}

	s.w.Header().Set("Content-Type", "application/json; charset=utf-8")
	s.w.Write(buf)
}

func (a *api) error(s *server, err error) {
	s.w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	s.w.WriteHeader(http.StatusInternalServerError)
	io.WriteString(s.w, err.Error())
}

// TODO(mdempsky): Unit tests to make sure queries below execute and
// produce valid JSON of the expected schema.

// TODO(mdempsky): The json_object argument lists are repetitive and
// somewhat redundant with the Go struct definitions. Can we use
// reflection to generate them automatically?

var apiAnnouncementList = api{
	value: func() interface{} { return new([]model.Announcement) },
	query: `
select json_arrayagg(json_object(
  'id',         a.id,
  'title',      a.title,
  'message',    a.message,
  'icon',       a.icon,
  'created_by', a.created_by,
  'send_time',  a.send_time,
  'sent',       a.sent != 0` /* TODO(mdempsky): Change SQL schema to use bool. */ + `
))
from announcements a
where a.sent
  and a.conference_id = :conference_id
`,
	args: func() interface{} {
		return new(struct {
			ConferenceID int `json:"conference_id" db:"conference_id"`
		})
	},
}

var apiConferenceList = api{
	value: func() interface{} { return new([]model.Conference) },
	query: `
select json_arrayagg(json_object(
  'id',         c.id,
  'name',       c.name,
  'start_date', c.start_date,
  'end_date',   c.end_date
))
from conferences c
where c.id = :conference_id
`,
	args: func() interface{} {
		return new(struct {
			ConferenceID int `json:"conference_id" db:"conference_id"`
		})
	},
}

var apiEventList = api{
	value: func() interface{} { return new([]model.Event) },
	query: `
select json_arrayagg(json_object(
  'id',            e.id,
  'name',          e.name,
  'description',   e.description,
  'start_time',    e.start_time,
  'length',        e.length,
  'key_event',     e.key_event != 0,` /* TODO(mdempsky): Change SQL schema to use bool */ + `
  'location',      json_object(
 		'name', l.name,
		'place_id', l.place_id,
		'address', l.address,
		'city', l.city,
		'lat', l.lat,
		'lng', l.lng
  ),
  'image_url',     e.image_url,
  'total_attendees', (
		select count(distinct rsvpTotal.user_id)
		from rsvp rsvpTotal
		where rsvpTotal.event_id = e.id and rsvpTotal.attending
  ),
  'attendees', (
		select json_arrayagg(json_object('name', users.name))
		from rsvp rsvpList
		join users on rsvpList.user_id = users.id
		where rsvpList.event_id = e.id and rsvpList.attending
  ),
  'attending', (
		case when(
			select attending
			from rsvp rsvpStatus
			where rsvpStatus.event_id = e.id and rsvpStatus.user_id = :user_id
		) then true
          else false
          end
  )
))
from events e
join locations l on e.location_id = l.id
where conference_id = :conference_id
`,
	args: func() interface{} {
		return new(struct {
			ConferenceID int `json:"conference_id" db:"conference_id"`
			UserID       int `json:"user_id" db:"user_id"`
		})
	},
}

var apiInfoList = api{
	value: func() interface{} { return new([]model.Info) },
	query: `
select json_arrayagg(json_object(
  'id',            i.id,
  'title',         i.title,
  'subtitle',      i.subtitle,
  'content',       i.content,
  'icon',          i.icon,
  'display_order', i.display_order
))
from info i
`,
}

var apiUserAdd = api{
	query: `
insert into users (conference_id, name, email, device_id, device_name, platform, timestamp)
values (:conference_id, :name, :email, :device_id, :device_name, :platform, now())
`,
	args: func() interface{} {
		return new(struct {
			ConferenceID   int    `json:"conference_id" db:"conference_id"`
			Name           string `json:"name" db:"name"`
			Email          string `json:"email" db:"email"`
			DeviceID       string `json:"device_id" db:"device_id"`
			DeviceName     string `json:"device_name" db:"device_name"`
			DevicePlatform string `json:"platform" db:"platform"`
		})
	},
}

var apiUserRSVP = api{
	query: `
replace into rsvp (event_id, user_id, attending, timestamp)
values (:event_id, (SELECT id FROM users WHERE device_id = :device_id), :attending, now())
`,
	args: func() interface{} {
		return new(struct {
			EventID   int    `json:"event_id" db:"event_id"`
			DeviceID  string `json:"device_id" db:"device_id"`
			Attending bool   `json:"attending" db:"attending"`
		})
	},
}

var apiUserRegisterPushNotifications = api{
	query: `
update users set expo_push_token = :expo_push_token where device_id = :device_id
`,
	args: func() interface{} {
		return new(struct {
			DeviceID      string `json:"device_id" db:"device_id"`
			ExpoPushToken string `json:"expo_push_token" db:"expo_push_token"`
		})
	},
}
