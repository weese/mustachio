package mustachio

import (
	"encoding/json"
	"testing"
)

func TestNumericIndexingAndDottedNames(t *testing.T) {
	template := "{{#recenttracks}}{{#track.0}}{{#@attr.nowplaying}}1{{{artist.#text}}} - {{{name}}}{{/@attr.nowplaying}}{{^@attr.nowplaying}}0{{/@attr.nowplaying}}{{/track.0}}{{/recenttracks}}"
	dataJSON := `{"recenttracks":{"track":[{"artist":{"mbid":"bca46a0c-25c9-42ca-98c2-e64c8a5e337e","#text":"Fred again.."},"streamable":"0","image":[{"size":"small","#text":"https:\/\/lastfm.freetls.fastly.net\/i\/u\/34s\/d9c66193ddb557f4d71c3ade3f6ac570.jpg"},{"size":"medium","#text":"https:\/\/lastfm.freetls.fastly.net\/i\/u\/64s\/d9c66193ddb557f4d71c3ade3f6ac570.jpg"},{"size":"large","#text":"https:\/\/lastfm.freetls.fastly.net\/i\/u\/174s\/d9c66193ddb557f4d71c3ade3f6ac570.jpg"},{"size":"extralarge","#text":"https:\/\/lastfm.freetls.fastly.net\/i\/u\/300x300\/d9c66193ddb557f4d71c3ade3f6ac570.jpg"}],"mbid":"","album":{"mbid":"","#text":"leavemealone"},"name":"leavemealone","@attr":{"nowplaying":"true"},"url":"https:\/\/www.last.fm\/music\/Fred+again..\/_\/leavemealone"},{"artist":{"mbid":"","#text":"Fred again.."},"streamable":"0","image":[{"size":"small","#text":"https:\/\/lastfm.freetls.fastly.net\/i\/u\/34s\/5e5eae8bd6beac6fa5597cbb87a2abc0.jpg"},{"size":"medium","#text":"https:\/\/lastfm.freetls.fastly.net\/i\/u\/64s\/5e5eae8bd6beac6fa5597cbb87a2abc0.jpg"},{"size":"large","#text":"https:\/\/lastfm.freetls.fastly.net\/i\/u\/174s\/5e5eae8bd6beac6fa5597cbb87a2abc0.jpg"},{"size":"extralarge","#text":"https:\/\/lastfm.freetls.fastly.net\/i\/u\/300x300\/5e5eae8bd6beac6fa5597cbb87a2abc0.jpg"}],"mbid":"","album":{"mbid":"","#text":"leavemealone (Nia Archives Remix)"},"name":"leavemealone - Nia Archives Remix","url":"https:\/\/www.last.fm\/music\/Fred+again..\/_\/leavemealone+-+Nia+Archives+Remix","date":{"uts":"1757256250","#text":"07 Sep 2025, 14:44"}}],"@attr":{"user":"foo","totalPages":"14042","page":"1","perPage":"1","total":"14042"}}}`
	var data map[string]any
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		t.Fatalf("json unmarshal error: %v", err)
	}
	out, err := Render(template, data, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "1Fred again.. - leavemealone" {
		t.Fatalf("got %q want %q", out, "1Fred again.. - leavemealone")
	}
}
