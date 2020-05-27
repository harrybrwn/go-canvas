package canvas

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/harrybrwn/errs"
	"github.com/matryer/is"
)

func testToken() string {
	tok := os.Getenv("CANVAS_TEST_TOKEN")
	if tok == "" {
		panic("no testing token")
	}
	return tok
}

func init() {
	t := testToken()
	SetToken(t)
}

var (
	mu            sync.Mutex
	testingUser   *User
	testingCourse *Course
)

func testUser() (*User, error) {
	var err error
	if testingUser == nil {
		testingUser, err = CurrentUser()
	}
	return testingUser, err
}

func testCourse() Course {
	if testingCourse == nil {
		var err error
		testingCourse, err = GetCourse(2056049)
		if err != nil {
			panic("could not get test course: " + err.Error())
		}
	}
	testingCourse.client = copydoer(testingCourse.client)
	return *testingCourse
}

type thing struct {
	ID int `json:"id"`
}

func (t *thing) id(s string) string {
	return fmt.Sprintf(s, t.ID)
}

func Test(t *testing.T) {
	th := thing{}
	b := []byte(`{"id": 123}`)
	json.Unmarshal(b, &th)
	fmt.Println(th)
	fmt.Println(th.id("/path/%d/to/thign"))
}

func TestAssignments(t *testing.T) {
	is := is.New(t)
	c := testCourse()
	i := 0
	for ass := range c.Assignments() {
		i++
		if ass.ID == 0 {
			t.Error("bad assignment id")
		}
	}
	if i != 1 {
		t.Error("should have one assignment")
	}

	tm := time.Now()
	newass, err := c.CreateAssignment(Assignment{
		Name:        "runtime test assignment",
		Description: "this is a test assignment that has been generated durning testing",
		DueAt:       &tm,
	})
	is.NoErr(err)
	if newass == nil {
		t.Fatal("new assignment is nil")
	}
	if newass.ID == 0 {
		t.Error("got a bad id, could not create assignment")
	}

	asses, err := c.ListAssignments(IncludeOpt("overrides"))
	is.NoErr(err)
	if len(asses) != 2 {
		t.Error("should have one assignment")
	}
	a, err := c.EditAssignment(&Assignment{ID: newass.ID, Name: "edited"})
	is.NoErr(err)
	is.Equal(a.Name, "edited")
	is.NoErr(errs.Eat(c.Assignment(newass.ID))) // i don't even need to test this but it makes my coverage better lol
	is.NoErr(errs.Eat(c.DeleteAssignment(newass)))
}

func TestSetHost(t *testing.T) {
	trans := defaultCanvas.client.Transport
	auth, ok := trans.(*auth)
	if !ok {
		t.Fatalf("could not set a host for this transport: %T", trans)
	}
	host := auth.host

	if err := SetHost("test.host"); err != nil {
		t.Error(err)
	}
	if auth.host != "test.host" {
		t.Error("did not set correct host")
	}
	defaultCanvas.client.Transport = http.DefaultTransport
	if err := SetHost("test1.host"); err == nil {
		t.Errorf("expected an error for setting host on %T", defaultCanvas.client.Transport)
	}
	defaultCanvas.client.Transport = auth
	auth.host = host
}

func TestAnnouncements(t *testing.T) {
	is := is.New(t)
	_, err := Announcements([]string{})
	is.True(err != nil)
	_, err = Announcements([]string{"course_1"})
	is.NoErr(err)
}

func TestCanvas_Err(t *testing.T) {
	for _, c := range []*Canvas{
		WithHost(testToken(), ""),
		WithHost("", DefaultHost),
	} {
		_, err := c.CurrentUser()
		if err == nil {
			t.Error("expected an error")
		}
		courses, err := c.Courses(ActiveCourses)
		if err == nil {
			t.Error("expected an error")
		}
		if courses != nil {
			t.Error("expected nil courses")
		}
	}
}

func TestUser_Err(t *testing.T) {
	is := is.New(t)
	u, err := testUser()
	is.NoErr(err)
	colors, err := u.Colors()
	is.NoErr(err)
	defer deauthorize(u.client)()

	settings, err := u.Settings()
	is.True(err != nil) // User.Settings should return an error when not authorized
	is.True(settings == nil)
	is.True(len(settings) == 0)

	profile, err := u.Profile()
	is.True(err != nil)
	is.True(profile == nil)

	var col string
	for col = range colors {
		break
	}
	color, err := u.Color(col)
	is.True(err != nil)
	is.True(color == nil)

	err = u.SetColor(col, "#FFFFFF")
	is.True(err != nil)
}

func TestSearchUser(t *testing.T) {
	c := testCourse()
	users, err := c.SearchUsers("test")
	if err != nil {
		t.Error(err)
	}
	if len(users) != 1 {
		t.Error("test account only has one user")
	}
	for _, u := range users {
		if u.Name != "Test User" {
			t.Error("wrong user")
		}
	}
}

func TestCourses(t *testing.T) {
	courses, err := Courses()
	if err != nil {
		t.Error(err)
	}
	for _, c := range courses {
		if c.ID == 0 {
			t.Error("bad course id")
		}
	}
}

func TestCourse_Settings(t *testing.T) {
	c := testCourse()
	settings, err := c.Settings()
	if err != nil {
		t.Error(err)
	}
	hidefinalgrades := settings.HideFinalGrades
	settings.HideFinalGrades = !hidefinalgrades
	settings, err = c.UpdateSettings(settings)
	if err != nil {
		t.Error(err)
	}
	if settings.HideFinalGrades == hidefinalgrades {
		t.Error("hide final grades should be the opposite")
	}
}

func TestCourse_Settings_Err(t *testing.T) {
	c := testCourse()
	defer deauthorize(c.client)()
	_, err := c.UpdateSettings(nil)
	if err == nil {
		t.Error("expected an error")
	}
}

func TestAccount(t *testing.T) {
	is := is.New(t)
	_, err := SearchAccounts("UC Berkeley")
	is.NoErr(err)

	t.Skip("can't figure out how to get account authorization")
	a, err := CurrentAccount()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(a)

	as, err := Accounts()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(as)
}

func TestBookmarks(t *testing.T) {
	is := is.New(t)

	c := testCourse()
	err := CreateBookmark(&Bookmark{
		Name: "test bookmark",
		URL:  fmt.Sprintf("https://%s/courses/%d/assignments", DefaultHost, c.ID),
	})
	if err != nil {
		t.Error(err)
	}

	bks, err := Bookmarks()
	is.NoErr(err)
	for _, b := range bks {
		if b.Name != "test bookmark" {
			t.Error("got the wrong bookmark")
		}
		is.NoErr(DeleteBookmark(&b))
	}

	defer deauthorize(defaultCanvas.client)()
	err = CreateBookmark(&Bookmark{
		Name: "test bookmark",
		URL:  fmt.Sprintf("https://%s/courses/%d/assignments", DefaultHost, c.ID),
	})
	if err == nil {
		t.Error("expected an error")
	}
}

func TestLinks(t *testing.T) {
	headers := []http.Header{
		{"Link": {`<https://canvas.instructure.com/api/v1/courses/000/users?search_term=test&page=1&per_page=10>; rel="current",<https://canvas.instructure.com/api/v1/courses/000/users?search_term=test&page=1&per_page=10>; rel="first",<https://canvas.instructure.com/api/v1/courses/000/users?search_term=test&page=45&per_page=10>; rel="last"`}},
		{"Link": {`<https://canvas.instructure.com/api/v1/courses/000/files?page=1&per_page=10>; rel="current",<https://canvas.instructure.com/api/v1/courses/000/files?page=2&per_page=10>; rel="next",<https://canvas.instructure.com/api/v1/courses/000/files?page=1&per_page=10>; rel="first",<https://canvas.instructure.com/api/v1/courses/000/files?page=45&per_page=10>; rel="last"`}},
	}
	for _, header := range headers {
		n, err := findlastpage(header)
		if err != nil {
			t.Error(err)
		}
		if n != 45 {
			t.Error("wrong page number")
		}
		links, err := newLinkedResource(header)
		if err != nil {
			t.Error(err)
		}
		if links.Last.page != 45 {
			t.Error("wrong page number")
		}
	}

	n, err := findlastpage(http.Header{})
	if err == nil {
		t.Error("expected an error")
	}
	if n != -1 {
		t.Error("n should be -1")
	}
}

func TestErrors(t *testing.T) {
	is := is.New(t)
	e := &AuthError{
		Status: "test",
		Errors: []errorMsg{{"one"}, {"two"}},
	}
	is.Equal(e.Error(), "test: one, two")
	e = &AuthError{
		Status: "",
		Errors: []errorMsg{{"one"}, {"two"}},
	}
	is.Equal(e.Error(), "one, two")
	is.Equal(checkErrors([]errorMsg{}), "")

	err := &Error{}
	json.Unmarshal([]byte(`{"errors":{"end_date":"no"},"message":"error"}`), err)
	is.Equal(err.Error(), "error")
	err = &Error{}
	json.Unmarshal([]byte(`{"errors":{"end_date":"no"}}`), err)
	is.Equal(err.Error(), "end_date: no")
	is.True(IsRateLimit(ErrRateLimitExceeded))
	is.True(!IsRateLimit(nil))
}

func TestOptions(t *testing.T) {
	is := is.New(t)
	o := ArrayOpt("include", "one", "two")
	o2 := IncludeOpt("one", "two")
	is.Equal(o.Name(), o2.Name())
	is.Equal(o.Value(), o2.Value())

	opts := []Option{
		Opt("key", "value"),
		DateOpt("date", time.Now()),
		SortOpt("date"),
	}
	q := optEnc(opts).Encode()
	if q == "" {
		t.Error("should not be empty")
	}
	if !strings.Contains(q, "sort") {
		t.Error("should have sorting option")
	}
	if !strings.Contains(q, "key=value") {
		t.Error("should have the key-value pair")
	}
	prefed := toPrefixedOpts("prefix", opts)
	for _, o := range prefed {
		if !strings.Contains(o.Name(), "prefix") {
			t.Error("should contain the prefix")
		}
	}
	options := optEnc{
		IncludeOpt("user"),
		IncludeOpt("another"),
	}
	if options.Encode() != "include%5B%5D=user&include%5B%5D=another" {
		t.Error("got wrong encoded value")
	}
	if (optEnc{}).Encode() != "" {
		t.Error("empty options should have empty encoded value")
	}
	o = UserOpt("key", "value")
	is.Equal(o.Name(), "user[key]")
	is.Equal(o.Value(), []string{"value"})
}

func deauthorize(d doer) (reset func()) {
	mu.Lock()
	defer mu.Unlock()
	reset = func() {
		fmt.Println("warning: client no deauthorized")
	}
	var cli *http.Client

	switch c := d.(type) {
	case *client:
		cli = &c.Client
	case *http.Client:
		cli = c
	default:
		return
	}
	var token string
	switch ath := cli.Transport.(type) {
	case *auth:
		token = ath.token
		ath.token = ""
		reset = func() { ath.token = token }
	case *syncAuth:
		token = ath.tok
		ath.tok = ""
		reset = func() { ath.tok = token }
	default:
		return
	}
	return reset
}

func copydoer(d doer) doer {
	if d == nil {
		return nil
	}
	cli := &http.Client{}
	switch dr := d.(type) {
	case *client:
		*cli = dr.Client
	case *http.Client:
		*cli = *dr
	default:
		panic(fmt.Sprintf("dont't know how to copy %T", d))
	}

	switch trans := cli.Transport.(type) {
	case *auth:
		a := &auth{}
		*a = *trans
		cli.Transport = a
	case *syncAuth:
		a := &syncAuth{}
		*a = *trans
		cli.Transport = a
	}
	return cli
}
