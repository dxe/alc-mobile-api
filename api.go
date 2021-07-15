package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/dxe/alc-mobile-api/model"
)

type api struct {
	// query is the underlying SQL query that the API issues to the
	// MySQL database. It should return a single-row, single-column JSON
	// result.
	query string

	// args is a struct that that the JSON request body is decoded into.
	args func() interface{}

	// value returns a pointer to a newly allocated Go variable able to
	// represent the JSON object returned by the query.
	value func() interface{}
}

// TODO: return an empty array instead of nothing if no results returned from database?

// TODO: if args are required but not supplied, return a specific error message?

func (a *api) serve(s *server) {
	args := a.args()
	err := json.NewDecoder(s.r.Body).Decode(&args)
	if err != nil && err != io.EOF {
		a.error(s, err)
		return
	}

	query, queryArgs, err := s.db.BindNamed(a.query, args)
	if err != nil {
		a.error(s, err)
		return
	}

	// TODO(mdempsky): Implement caching and/or single-flighting (e.g.,
	// golang.org/x/sync/singleflight), so we don't need to issue a DB
	// request for each HTTP request.

	var buf []byte
	if err := s.db.QueryRowxContext(s.r.Context(), query, queryArgs...).Scan(&buf); err != nil {
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
`,
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
  'location_id',   e.location_id,
  'image_url',     e.image_url
))
from events e
where conference_id = :conference_id
`,
	args: func() interface{} {
		return new(struct {
			ConferenceID int `json:"conference_id" db:"conference_id"`
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

func (a *api) error(s *server, err error) {
	s.w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	s.w.WriteHeader(http.StatusInternalServerError)
	io.WriteString(s.w, err.Error())
}
