package canvas

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
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

func Test(t *testing.T) {}

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

	now := time.Now().UTC()
	newass, err := c.CreateAssignment(Assignment{
		Name:        "runtime test assignment",
		Description: "this is a test assignment that has been generated durning testing",
		DueAt:       now,
	})
	is.NoErr(err)
	if newass == nil {
		t.Fatal("new assignment is nil")
	}
	if newass.ID == 0 {
		t.Error("got a bad id, could not create assignment")
	}
	now = now.Round(time.Second) // canvas' servers round to the second
	// Sometimes the time given back is off by one second
	if !(newass.DueAt.Equal(now) || newass.DueAt.Add(time.Second).Equal(now)) {
		t.Errorf("due date should not have changed after response; got %v, want %v", newass.DueAt, now)
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
	code := fmt.Sprintf("course_%d", testCourse().ID)
	_, err = Announcements([]string{code})
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

func TestCalendarEvents(t *testing.T) {
	course := testCourse()
	contextCode := fmt.Sprintf("course_%d", course.ID)
	now := time.Now().UTC()
	event, err := CreateCalendarEvent(&CalendarEvent{
		Title:       "test event",
		Description: "this is a test event and should not exists, please delete me",
		StartAt:     now,
		AllDay:      true,
		ContextCode: contextCode,
	})
	if err != nil {
		t.Error(err)
	}
	calendar, err := CalendarEvents(ArrayOpt("context_codes", contextCode))
	if err != nil {
		t.Error(err)
	}
	i := 0
	for range calendar {
		i++
	}
	if i < 1 {
		t.Errorf("should have at least one calendar event, got %d", i)
	}
	_, err = DeleteCalendarEvent(event)
	if err != nil {
		t.Error(err)
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
	ConcurrentErrorHandler = func(e error) error {
		if e == nil {
			t.Error("expected an error")
		}
		return e
	}
	i := 0
	for f := range JoinFileObjs(u.Files(), u.Folders()) {
		if f.GetID() == 0 {
			t.Error("got bad id")
		}
		i++
	}
	if i != 0 {
		t.Error("should not have gotten any files")
	}
	ConcurrentErrorHandler = defaultErrorHandler
}

func TestUser(t *testing.T) {
	is := is.New(t)
	client, mux, server := testServer()
	defer server.Close()
	defer swapCanvas(&Canvas{client: client})()
	nfiles := 6
	mux.HandleFunc("/api/v1/users/2", func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		w.WriteHeader(200)
		writeTestFile(t, "user.json", w)
	})
	mux.HandleFunc("/api/v1/users/2/files", filesHandlerFunc(t, nfiles))
	mux.HandleFunc("/api/v1/users/2/folders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			is.Equal(r.URL.Query().Get("name"), "tests")
			writeTestFile(t, "folder.json", w)
		case "GET":
			fn := foldersHandlerFunc(t, nfiles)
			fn(w, r)
		}
	})
	user, err := GetUser(2)
	is.NoErr(err)
	is.Equal(user.ID, 2)
	i := 0
	for f := range user.Files() {
		i++
		is.Equal(f.ID, 569)
	}
	is.Equal(i, nfiles)
	files, err := user.ListFiles()
	is.NoErr(err)
	is.Equal(len(files), nfiles)
	folder, err := user.CreateFolder("tests")
	if err != nil {
		t.Error(err)
	}
	is.True(folder != nil)
	folders, err := user.ListFolders()
	is.NoErr(err)
	is.Equal(len(folders), nfiles)
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

func TestCourseFileObjects(t *testing.T) {
	c := testCourse()
	folder, err := c.CreateFolder(path.Join("/", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	if err = folder.Rename(t.Name()); err != nil {
		t.Error(err)
	}
	if folder.Foldername != t.Name() {
		t.Errorf("could not rename the new file to %s", t.Name())
	}

	for f := range JoinFileObjs(c.Files(), c.Folders()) {
		if f.GetID() == 0 {
			t.Error("got a zero id")
		}
		if f.Name() == "" {
			t.Error("got an empty name")
		}
	}
	list, err := c.ListFiles(Opt("search_term", "main.go"))
	if err != nil {
		t.Error(err)
	}
	if len(list) < 1 {
		t.Error("could not find main.go")
	}
	if err = list[0].Copy(folder); err != nil {
		t.Error(err)
	}
	if err = folder.Delete(Opt("force", true)); err != nil {
		t.Error(err)
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
	is.NoErr(err)
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

func TestCourse_User(t *testing.T) {
	client, mux, server := testServer()
	defer server.Close()
	defer swapCanvas(&Canvas{client: client})()
	mux.HandleFunc("/api/v1/courses/1234/users/2", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		assertMethod(t, r, "GET")
		writeTestFile(t, "user.json", w)
	})
	course := &Course{client: client, ID: 1234}
	user, err := course.User(2)
	if err != nil {
		t.Fatal(err)
	}

	if user.ID != 2 {
		t.Error("wrong id")
	}
}

func TestCourse_DiscussionTopics(t *testing.T) {
	c := testCourse()
	discs, err := c.DiscussionTopics()
	if err != nil {
		t.Error(err)
	}
	for _, d := range discs {
		if d.ID == 0 {
			t.Error("got zero id")
		}
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

func TestRateLimitErr(t *testing.T) {
	cli, mux, server := testServer()
	defer server.Close()
	defer swapCanvas(&Canvas{client: cli})()
	mux.HandleFunc("/api/v1/accounts/self", func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		w.WriteHeader(http.StatusForbidden)
	})
	mux.HandleFunc("/api/v1/accounts", func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		w.WriteHeader(http.StatusForbidden)
	})
	mux.HandleFunc("/api/v1/folders/123/copy_file", func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "POST")
		q := r.URL.Query()
		if q.Get("source_file_id") != "54321" {
			t.Error("wrong source_file_id")
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
	})
	_, err := CurrentAccount()
	if !IsRateLimit(err) {
		t.Error("expected rate limit error")
	}
	_, err = Accounts()
	if !IsRateLimit(err) {
		t.Error("expected rate limit error")
	}
	folder := &Folder{ID: 123}
	file := &File{client: cli, ID: 54321}
	err = file.Copy(folder)
	if err == nil {
		t.Error("expected an error")
	}
}

func TestUtils(t *testing.T) {
	tests := []struct {
		in, out string
	}{
		{"Course", "courses"},
		{"User", "users"},
		{"GroupCategory", "group_categories"},
		{"Account", "accounts"},
	}
	for _, test := range tests {
		if path := pathFromContextType(test.in); path != test.out {
			t.Errorf("got %s; wanted %s", path, test.out)
		}
	}
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

func testServer() (*http.Client, *http.ServeMux, *httptest.Server) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	transport := &TestingTransport{&http.Transport{
		Proxy: func(r *http.Request) (*url.URL, error) { return url.Parse(server.URL) },
	}}
	client := &http.Client{Transport: transport}
	authorize(client, "", DefaultHost)
	return client, mux, server
}

type TestingTransport struct {
	transport http.RoundTripper
}

func (tt *TestingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	return tt.transport.RoundTrip(req)
}

func writeTestFile(t *testing.T, file string, w io.Writer) {
	t.Helper()
	b, err := ioutil.ReadFile(fmt.Sprintf("./testdata/%s", file))
	if err != nil {
		t.Error("could not read testdata")
	}
	if _, err = w.Write(b); err != nil {
		t.Error("could not write test data:", err)
	}
}

func swapCanvas(c *Canvas) func() {
	reset := defaultCanvas
	defaultCanvas = c
	return func() {
		defaultCanvas = reset
	}
}

func assertMethod(t *testing.T, r *http.Request, method string) {
	t.Helper()
	if r.Method != method {
		t.Errorf("wrong method: expected %s; got %s", method, r.Method)
	}
}

func TestCurrentUser(t *testing.T) {
	cli, mux, server := testServer()
	defer server.Close()
	mux.HandleFunc("/api/v1/users/self", func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		writeTestFile(t, "user.json", w)
	})
	canv := &Canvas{client: cli}
	u, err := canv.CurrentUser()
	if err != nil {
		t.Error("could not get user:", err)
	}
	if u == nil {
		t.Error("user is nil")
	}
	if u.ID != 2 {
		t.Error("the testing user should have id == 2, not ", u.ID)
	}
	if u.client != cli {
		t.Error("didn't pass the client along")
	}
}
