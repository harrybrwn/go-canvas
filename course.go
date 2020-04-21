package canvas

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"sync"
	"time"
)

// Course represents a canvas course.
type Course struct {
	ID                   int         `json:"id"`
	Name                 string      `json:"name"`
	SisCourseID          int         `json:"sis_course_id"`
	UUID                 string      `json:"uuid"`
	IntegrationID        interface{} `json:"integration_id"`
	SisImportID          int         `json:"sis_import_id"`
	CourseCode           string      `json:"course_code"`
	WorkflowState        string      `json:"workflow_state"`
	AccountID            int         `json:"account_id"`
	RootAccountID        int         `json:"root_account_id"`
	EnrollmentTermID     int         `json:"enrollment_term_id"`
	GradingStandardID    int         `json:"grading_standard_id"`
	GradePassbackSetting string      `json:"grade_passback_setting"`
	CreatedAt            time.Time   `json:"created_at"`
	StartAt              time.Time   `json:"start_at"`
	EndAt                time.Time   `json:"end_at"`
	Locale               string      `json:"locale"`
	Enrollments          []struct {
		EnrollmentState                string `json:"enrollment_state"`
		Role                           string `json:"role"`
		RoleID                         int64  `json:"role_id"`
		Type                           string `json:"type"`
		UserID                         int64  `json:"user_id"`
		LimitPrivilegesToCourseSection bool   `json:"limit_privileges_to_course_section"`
	} `json:"enrollments"`
	TotalStudents     int         `json:"total_students"`
	Calendar          interface{} `json:"calendar"`
	DefaultView       string      `json:"default_view"`
	SyllabusBody      string      `json:"syllabus_body"`
	NeedsGradingCount int         `json:"needs_grading_count"`

	Term           Term           `json:"term"`
	CourseProgress CourseProgress `json:"course_progress"`

	ApplyAssignmentGroupWeights bool `json:"apply_assignment_group_weights"`
	Permissions                 struct {
		CreateDiscussionTopic bool `json:"create_discussion_topic"`
		CreateAnnouncement    bool `json:"create_announcement"`
	} `json:"permissions"`
	IsPublic                         bool   `json:"is_public"`
	IsPublicToAuthUsers              bool   `json:"is_public_to_auth_users"`
	PublicSyllabus                   bool   `json:"public_syllabus"`
	PublicSyllabusToAuth             bool   `json:"public_syllabus_to_auth"`
	PublicDescription                string `json:"public_description"`
	StorageQuotaMb                   int    `json:"storage_quota_mb"`
	StorageQuotaUsedMb               int    `json:"storage_quota_used_mb"`
	HideFinalGrades                  bool   `json:"hide_final_grades"`
	License                          string `json:"license"`
	AllowStudentAssignmentEdits      bool   `json:"allow_student_assignment_edits"`
	AllowWikiComments                bool   `json:"allow_wiki_comments"`
	AllowStudentForumAttachments     bool   `json:"allow_student_forum_attachments"`
	OpenEnrollment                   bool   `json:"open_enrollment"`
	SelfEnrollment                   bool   `json:"self_enrollment"`
	RestrictEnrollmentsToCourseDates bool   `json:"restrict_enrollments_to_course_dates"`
	CourseFormat                     string `json:"course_format"`
	AccessRestrictedByDate           bool   `json:"access_restricted_by_date"`
	TimeZone                         string `json:"time_zone"`
	Blueprint                        bool   `json:"blueprint"`
	BlueprintRestrictions            struct {
		Content           bool `json:"content"`
		Points            bool `json:"points"`
		DueDates          bool `json:"due_dates"`
		AvailabilityDates bool `json:"availability_dates"`
	} `json:"blueprint_restrictions"`
	BlueprintRestrictionsByObjectType struct {
		Assignment struct {
			Content bool `json:"content"`
			Points  bool `json:"points"`
		} `json:"assignment"`
		WikiPage struct {
			Content bool `json:"content"`
		} `json:"wiki_page"`
	} `json:"blueprint_restrictions_by_object_type"`

	client *client
}

// Files returns a list of all the courses files
func (c *Course) Files() ([]*File, error) {
	path := fmt.Sprintf("courses/%d/files", c.ID)
	files := make([]*File, 0)

	resp, err := c.client.get(path, url.Values{
		"sort":     {"created_at"},
		"per_page": {"10"},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, err
	}

	for i := range files {
		files[i].client = c.client
	}
	return files, nil
}

// FilesChan returns a channel the streams course files.
func (c *Course) FilesChan() <-chan *File {
	files := make(chan *File)
	filesCh, errs := c.files()

	go func() {
		for {
			select {
			case err := <-errs:
				if err != nil {
					panic(err)
				}
			case f := <-filesCh:
				if f == nil {
					close(files)
					return
				}
				files <- f
			default:
			}
		}
	}()
	return files
}

func (c *Course) files() (<-chan *File, <-chan error) {
	var (
		path  = fmt.Sprintf("courses/%d/files", c.ID)
		files = make(chan *File)
		errs  = make(chan error)
		wg    sync.WaitGroup
	)

	// First we get page 1 and store the "Link" header value
	// so that we know how many pages there are (see newLinkedResource).
	resp, err := c.client.get(path, url.Values{
		"page": {"1"},
	})
	if err != nil {
		errs <- err
		return nil, errs
	}

	// The is where we store the links. We do this so that
	// we know how many pages there are.
	pages, err := newLinkedResource(resp)
	if err != nil {
		errs <- err
		return nil, errs
	}
	lastpage, ok := pages.links["last"]
	if !ok {
		errs <- errors.New("could not find last page")
		return nil, errs
	}
	n := lastpage.page // number of pages
	wg.Add(n)
	// send files from first page
	go func() {
		// Also, since we have already made the request for the first page,
		// we may as well decode the files and send them in the channel.
		pageOneFiles, err := decodeAndCloseFiles(resp.Body)
		if err != nil {
			errs <- err
		}
		for _, f := range pageOneFiles {
			files <- f
		}
		wg.Done()
	}()
	// get the rest of the pages and send
	for page := 2; page <= n; page++ {
		go c.asyncGetFiles(path, page, files, errs, &wg)
	}
	go func() {
		wg.Wait()
		close(files)
		close(errs)
	}()
	return files, errs
}

func (c Course) asyncGetFiles(path string, page int, files chan<- *File, errs chan<- error, wg *sync.WaitGroup) error {
	defer wg.Done()
	resp, err := c.client.get(path, url.Values{
		"page": {strconv.FormatInt(int64(page), 10)},
	})
	if err != nil {
		errs <- err
		return err
	}
	defer resp.Body.Close()
	arr := make([]*File, 0, 10)
	if err = json.NewDecoder(resp.Body).Decode(&arr); err != nil {
		errs <- err
		return err
	}
	for _, f := range arr {
		files <- f
	}
	return nil
}

// CourseOption is a string type that defines the available course options.
type CourseOption string

const (
	// NeedsGradingCountOpt is a course option
	NeedsGradingCountOpt CourseOption = "needs_grading_count"
	// SyllabusBodyOpt is a course option
	SyllabusBodyOpt CourseOption = "syllabus_body"
	// PublicDescriptionOpt is a course option
	PublicDescriptionOpt CourseOption = "public_description"
	// TotalScoresOpt is a course option
	TotalScoresOpt CourseOption = "total_scores"
	// CurrentGradingPeriodScoresOpt is a course option
	CurrentGradingPeriodScoresOpt CourseOption = "current_grading_period_scores"
	// TermOpt is a course option
	TermOpt CourseOption = "term"
	// AccountOpt is a course option
	AccountOpt CourseOption = "account"
	// CourseProgressOpt is a course option
	CourseProgressOpt CourseOption = "course_progress"
	// SectionsOpt is a course option
	SectionsOpt CourseOption = "sections"
	// StorageQuotaUsedMBOpt is a course option
	StorageQuotaUsedMBOpt CourseOption = "storage_quota_used_mb"
	// TotalStudentsOpt is a course option
	TotalStudentsOpt CourseOption = "total_students"
	// PassbackStatusOpt is a course option
	PassbackStatusOpt CourseOption = "passback_status"
	// FavoritesOpt is a course option
	FavoritesOpt CourseOption = "favorites"
	// TeachersOpt is a course option
	TeachersOpt CourseOption = "teachers"
	// ObservedUsersOpt is a course option
	ObservedUsersOpt CourseOption = "observed_users"
	// CourseImageOpt is a course option
	CourseImageOpt CourseOption = "course_image"
	// ConcludedOpt is a course option
	ConcludedOpt CourseOption = "concluded"
)

func (opt CourseOption) String() string {
	return string(opt)
}

// Term is a school term. One school year.
type Term struct {
	ID      int
	Name    string
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
}

// CourseProgress is the progress through a course.
type CourseProgress struct {
	RequirementCount          int    `json:"requirement_count"`
	RequirementCompletedCount int    `json:"requirement_completed_count"`
	NextRequirementURL        string `json:"next_requirement_url"`
	CompletedAt               string `json:"completed_at"`
}

// Enrollment is an enrollment object
type Enrollment struct {
	ID                             int         `json:"id"`
	CourseID                       int         `json:"course_id"`
	SisCourseID                    string      `json:"sis_course_id"`
	CourseIntegrationID            string      `json:"course_integration_id"`
	CourseSectionID                int         `json:"course_section_id"`
	SectionIntegrationID           string      `json:"section_integration_id"`
	SisAccountID                   string      `json:"sis_account_id"`
	SisSectionID                   string      `json:"sis_section_id"`
	SisUserID                      string      `json:"sis_user_id"`
	EnrollmentState                string      `json:"enrollment_state"`
	LimitPrivilegesToCourseSection bool        `json:"limit_privileges_to_course_section"`
	SisImportID                    int         `json:"sis_import_id"`
	RootAccountID                  int         `json:"root_account_id"`
	Type                           string      `json:"type"`
	UserID                         int         `json:"user_id"`
	AssociatedUserID               interface{} `json:"associated_user_id"`
	Role                           string      `json:"role"`
	RoleID                         int         `json:"role_id"`
	CreatedAt                      time.Time   `json:"created_at"`
	UpdatedAt                      time.Time   `json:"updated_at"`
	StartAt                        time.Time   `json:"start_at"`
	EndAt                          time.Time   `json:"end_at"`
	LastActivityAt                 time.Time   `json:"last_activity_at"`
	LastAttendedAt                 time.Time   `json:"last_attended_at"`
	TotalActivityTime              int         `json:"total_activity_time"`
	HTMLURL                        string      `json:"html_url"`
	Grades                         struct {
		HTMLURL              string  `json:"html_url"`
		CurrentScore         string  `json:"current_score"`
		CurrentGrade         string  `json:"current_grade"`
		FinalScore           float64 `json:"final_score"`
		FinalGrade           string  `json:"final_grade"`
		UnpostedCurrentGrade string  `json:"unposted_current_grade"`
		UnpostedFinalGrade   string  `json:"unposted_final_grade"`
		UnpostedCurrentScore string  `json:"unposted_current_score"`
		UnpostedFinalScore   string  `json:"unposted_final_score"`
	} `json:"grades"`
	User struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		SortableName string `json:"sortable_name"`
		ShortName    string `json:"short_name"`
	} `json:"user"`
	OverrideGrade                     string  `json:"override_grade"`
	OverrideScore                     float64 `json:"override_score"`
	UnpostedCurrentGrade              string  `json:"unposted_current_grade"`
	UnpostedFinalGrade                string  `json:"unposted_final_grade"`
	UnpostedCurrentScore              string  `json:"unposted_current_score"`
	UnpostedFinalScore                string  `json:"unposted_final_score"`
	HasGradingPeriods                 bool    `json:"has_grading_periods"`
	TotalsForAllGradingPeriodsOption  bool    `json:"totals_for_all_grading_periods_option"`
	CurrentGradingPeriodTitle         string  `json:"current_grading_period_title"`
	CurrentGradingPeriodID            int     `json:"current_grading_period_id"`
	CurrentPeriodOverrideGrade        string  `json:"current_period_override_grade"`
	CurrentPeriodOverrideScore        float64 `json:"current_period_override_score"`
	CurrentPeriodUnpostedCurrentScore float64 `json:"current_period_unposted_current_score"`
	CurrentPeriodUnpostedFinalScore   float64 `json:"current_period_unposted_final_score"`
	CurrentPeriodUnpostedCurrentGrade string  `json:"current_period_unposted_current_grade"`
	CurrentPeriodUnpostedFinalGrade   string  `json:"current_period_unposted_final_grade"`
}

func (c *Course) path(p ...string) string {
	return path.Join(p...)
}

func (c *Course) setClient(cl *client) {
	c.client = cl
}

func decodeAndCloseFiles(rc io.ReadCloser) ([]*File, error) {
	files := make([]*File, 0)
	var err error
	defer func() {
		e := rc.Close()
		if err == nil {
			err = e
		}
	}()
	if err = json.NewDecoder(rc).Decode(&files); err != nil {
		return nil, err
	}
	return files, err
}

var resourceRegex = regexp.MustCompile(`<(.*?)>; rel="(.*?)"`)

func newLinkedResource(rsp *http.Response) (*linkedResource, error) {
	var err error
	resource := &linkedResource{
		resp:  rsp,
		links: map[string]*link{},
	}
	links := rsp.Header.Get("Link")
	parts := resourceRegex.FindAllStringSubmatch(links, -1)

	for _, part := range parts {
		resource.links[part[2]], err = newlink(part[1])
		if err != nil {
			return resource, err
		}
	}
	return resource, nil
}

type linkedResource struct {
	resp  *http.Response
	links map[string]*link
}

type link struct {
	url  *url.URL
	page int
}

func newlink(urlstr string) (*link, error) {
	u, err := url.Parse(urlstr)
	if err != nil {
		return nil, err
	}
	page, err := strconv.ParseInt(u.Query().Get("page"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("could not parse page num: %w", err)
	}
	return &link{
		url:  u,
		page: int(page),
	}, nil
}
