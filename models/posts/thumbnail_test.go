package posts_test

import (
	"testing"

	"github.com/kirsle/blog/models/posts"
)

func TestThumbnailRegexp(t *testing.T) {
	type testCase struct {
		Text       string
		Expect     string
		ExpectFail bool
	}

	var tests = []testCase{
		{
			Text:       "Hello world",
			ExpectFail: true,
		},
		{
			Text: "Some text.\n\n![An image](/static/photos/Image-1.jpg)\n" +
				"![Another image](/static/photos/Image-2.jpg)",
			Expect: "/static/photos/Image-1.jpg",
		},
		{
			Text: `<a href="/static/photos/12Abc456.jpg" target="_blank">` +
				`<img src="/static/photos/34Xyz123.jpg"></a>`,
			Expect: "/static/photos/12Abc456.jpg",
		},
		{
			Text: `A markdown image: ![With text](/test1.gif) and an HTML ` +
				`image: <img src="/test2.png">`,
			Expect: "/test1.gif",
		},
		{
			Text:   `<a href="https://google.com/"><img src="https://example.com/logo.gif?query=string.jpg"></a>`,
			Expect: "https://example.com/logo.gif?query=string.jpg",
		},
	}
	for _, test := range tests {
		p := &posts.Post{
			Body: test.Text,
		}

		result, ok := p.ExtractThumbnail()
		if !ok && !test.ExpectFail {
			t.Errorf("Text: %s\nExpected to fail, but did not!\nGot: %s",
				test.Text,
				result,
			)
			continue
		}

		if result != test.Expect {
			t.Errorf("Text: %s\nExpect: %s\nGot: %s",
				test.Text,
				test.Expect,
				result,
			)
		}
	}
}
