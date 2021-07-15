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

	// value returns a pointer to a newly allocated Go variable able to
	// represent the JSON object returned by the query.
	value func() interface{}
}

func (a *api) serve(s *server) {
	// TODO(mdempsky): Add mechanism for queries to specify custom
	// arguments picked out of the request.

	// TODO(mdempsky): Implement caching and/or single-flighting (e.g.,
	// golang.org/x/sync/singleflight), so we don't need to issue a DB
	// request for each HTTP request.

	var buf []byte
	if err := s.db.QueryRowContext(s.r.Context(), a.query).Scan(&buf); err != nil {
		s.w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		s.w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(s.w, err.Error())
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
  and a.conference_id = 1
`,
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
  'location',      json_object(
 		'name', l.name,
		'place_id', l.place_id,
		'address', l.address,
		'city', l.city,
		'lat', l.lat,
		'lng', l.lng
  ),
  'image_url',     e.image_url
))
from events e
join locations l on e.location_id = l.id
`,
	// TODO(mdempsky): Filter down conferences?
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
