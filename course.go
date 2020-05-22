package canvas

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/harrybrwn/errs"
)

// Course represents a canvas course.
type Course struct {
	ID                   int       `json:"id"`
	Name                 string    `json:"name"`
	SisCourseID          int       `json:"sis_course_id"`
	UUID                 string    `json:"uuid"`
	IntegrationID        string    `json:"integration_id"`
	SisImportID          int       `json:"sis_import_id"`
	CourseCode           string    `json:"course_code"`
	WorkflowState        string    `json:"workflow_state"`
	AccountID            int       `json:"account_id"`
	RootAccountID        int       `json:"root_account_id"`
	EnrollmentTermID     int       `json:"enrollment_term_id"`
	GradingStandardID    int       `json:"grading_standard_id"`
	GradePassbackSetting string    `json:"grade_passback_setting"`
	CreatedAt            time.Time `json:"created_at"`
	StartAt              time.Time `json:"start_at"`
	EndAt                time.Time `json:"end_at"`
	Locale               string    `json:"locale"`
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

	client       doer
	errorHandler func(error)
}

// Settings gets the course settings
func (c *Course) Settings(opts ...Option) (cs *CourseSettings, err error) {
	cs = &CourseSettings{}
	return cs, getjson(c.client, cs, asParams(opts), "/courses/%d/settings", c.ID)
}

// UpdateSettings will update a user's settings based on a given settings struct and
// will return the updated settings struct.
func (c *Course) UpdateSettings(settings *CourseSettings) (*CourseSettings, error) {
	m := make(map[string]interface{})
	raw, err := json.Marshal(settings)
	if err = errs.Pair(err, json.Unmarshal(raw, &m)); err != nil {
		return nil, err
	}

	vals := make(params)
	for k, v := range m {
		vals[k] = []string{fmt.Sprintf("%v", v)}
	}
	resp, err := put(c.client, fmt.Sprintf("/courses/%d/settings", c.ID), vals)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	s := CourseSettings{}
	return &s, json.NewDecoder(resp.Body).Decode(&s)
}

// CourseSettings is a json struct for a course's settings.
type CourseSettings struct {
	AllowStudentDiscussionTopics  bool `json:"allow_student_discussion_topics"`
	AllowStudentForumAttachments  bool `json:"allow_student_forum_attachments"`
	AllowStudentDiscussionEditing bool `json:"allow_student_discussion_editing"`
	GradingStandardEnabled        bool `json:"grading_standard_enabled"`
	GradingStandardID             int  `json:"grading_standard_id"`
	AllowStudentOrganizedGroups   bool `json:"allow_student_organized_groups"`
	HideFinalGrades               bool `json:"hide_final_grades"`
	HideDistributionGraphs        bool `json:"hide_distribution_graphs"`
	LockAllAnnouncements          bool `json:"lock_all_announcements"`
	UsageRightsRequired           bool `json:"usage_rights_required"`
}

// Users will get a list of users in the course
func (c *Course) Users(opts ...Option) (users []*User, err error) {
	return c.collectUsers("/courses/%d/users", opts)
}

// SearchUsers will search for a user in the course
func (c *Course) SearchUsers(term string, opts ...Option) (users []*User, err error) {
	opts = append(opts, Opt("search_term", term))
	return c.collectUsers("/courses/%d/search_users", opts)
}

// User gets a specific user.
func (c *Course) User(id int, opts ...Option) (*User, error) {
	u := &User{client: c.client}
	return u, getjson(c.client, u, asParams(opts), "/courses/%d/users/%d", c.ID, id)
}

// Assignment will get an assignment from the course given an id.
func (c *Course) Assignment(id int, opts ...Option) (ass *Assignment, err error) {
	return ass, getjson(
		c.client, &ass,
		asParams(opts),
		"/courses/%d/assignments/%d", c.ID, id,
	)
}

// Assignments send the courses assignments over a channel concurrently.
func (c *Course) Assignments(opts ...Option) <-chan *Assignment {
	ch := make(assignmentChan)
	pages := c.assignmentspager(ch, opts)
	go handleErrs(pages.start(), ch, c.errorHandler)
	return ch
}

// ListAssignments will get all the course assignments and put them in a slice.
func (c *Course) ListAssignments(opts ...Option) (asses []*Assignment, err error) {
	ch := make(assignmentChan)
	pages := c.assignmentspager(ch, opts)
	errs := pages.start()
	for {
		select {
		case as := <-ch:
			asses = append(asses, as)
		case err = <-errs:
			return asses, err
		}
	}
}

// Assignment is a struct holding assignment data
type Assignment struct {
	ID                             int              `json:"id"`
	Name                           string           `json:"name"`
	Description                    string           `json:"description"`
	CreatedAt                      string           `json:"created_at"`
	UpdatedAt                      string           `json:"updated_at"`
	DueAt                          string           `json:"due_at"`
	LockAt                         string           `json:"lock_at"`
	UnlockAt                       string           `json:"unlock_at"`
	HasOverrides                   bool             `json:"has_overrides"`
	AllDates                       interface{}      `json:"all_dates"`
	CourseID                       int              `json:"course_id"`
	HTMLURL                        string           `json:"html_url"`
	SubmissionsDownloadURL         string           `json:"submissions_download_url"`
	AssignmentGroupID              int              `json:"assignment_group_id"`
	DueDateRequired                bool             `json:"due_date_required"`
	AllowedExtensions              []string         `json:"allowed_extensions"`
	MaxNameLength                  int              `json:"max_name_length"`
	TurnitinEnabled                bool             `json:"turnitin_enabled"`
	VericiteEnabled                bool             `json:"vericite_enabled"`
	TurnitinSettings               TurnitinSettings `json:"turnitin_settings"`
	GradeGroupStudentsIndividually bool             `json:"grade_group_students_individually"`
	ExternalToolTagAttributes      interface{}      `json:"external_tool_tag_attributes"`
	PeerReviews                    bool             `json:"peer_reviews"`
	AutomaticPeerReviews           bool             `json:"automatic_peer_reviews"`
	PeerReviewCount                int              `json:"peer_review_count"`
	PeerReviewsAssignAt            string           `json:"peer_reviews_assign_at"`
	IntraGroupPeerReviews          bool             `json:"intra_group_peer_reviews"`
	GroupCategoryID                int              `json:"group_category_id"`
	NeedsGradingCount              int              `json:"needs_grading_count"`
	NeedsGradingCountBySection     []struct {
		SectionID         string `json:"section_id"`
		NeedsGradingCount int    `json:"needs_grading_count"`
	} `json:"needs_grading_count_by_section"`
	Position        int    `json:"position"`
	PostToSis       bool   `json:"post_to_sis"`
	IntegrationID   string `json:"integration_id"`
	IntegrationData struct {
		Num5678 string `json:"5678"`
	} `json:"integration_data"`
	PointsPossible          float64         `json:"points_possible"`
	SubmissionTypes         []string        `json:"submission_types"`
	HasSubmittedSubmissions bool            `json:"has_submitted_submissions"`
	GradingType             string          `json:"grading_type"`
	GradingStandardID       interface{}     `json:"grading_standard_id"`
	Published               bool            `json:"published"`
	Unpublishable           bool            `json:"unpublishable"`
	OnlyVisibleToOverrides  bool            `json:"only_visible_to_overrides"`
	LockedForUser           bool            `json:"locked_for_user"`
	LockInfo                LockInfo        `json:"lock_info"`
	LockExplanation         string          `json:"lock_explanation"`
	QuizID                  int             `json:"quiz_id"`
	AnonymousSubmissions    bool            `json:"anonymous_submissions"`
	DiscussionTopic         DiscussionTopic `json:"discussion_topic"`
	FreezeOnCopy            bool            `json:"freeze_on_copy"`
	Frozen                  bool            `json:"frozen"`
	FrozenAttributes        []string        `json:"frozen_attributes"`
	Submission              interface{}     `json:"submission"` // TODO: create a Submission struct and set this type to that
	UseRubricForGrading     bool            `json:"use_rubric_for_grading"`

	// RubricSettings                  string          `json:"rubric_settings"`
	RubricSettings interface{} `json:"rubric_settings"`

	Rubric []RubricCriteria `json:"rubric"`

	AssignmentVisibility            []int       `json:"assignment_visibility"`
	Overrides                       interface{} `json:"overrides"`
	OmitFromFinalGrade              bool        `json:"omit_from_final_grade"`
	ModeratedGrading                bool        `json:"moderated_grading"`
	GraderCount                     int         `json:"grader_count"`
	FinalGraderID                   int         `json:"final_grader_id"`
	GraderCommentsVisibleToGraders  bool        `json:"grader_comments_visible_to_graders"`
	GradersAnonymousToGraders       bool        `json:"graders_anonymous_to_graders"`
	GraderNamesVisibleToFinalGrader bool        `json:"grader_names_visible_to_final_grader"`
	AnonymousGrading                bool        `json:"anonymous_grading"`
	AllowedAttempts                 int         `json:"allowed_attempts"`
	PostManually                    bool        `json:"post_manually"`
}

// TurnitinSettings is a settings struct for turnitin
type TurnitinSettings struct {
	OriginalityReportVisibility string `json:"originality_report_visibility"`
	SPaperCheck                 bool   `json:"s_paper_check"`
	InternetCheck               bool   `json:"internet_check"`
	JournalCheck                bool   `json:"journal_check"`
	ExcludeBiblio               bool   `json:"exclude_biblio"`
	ExcludeQuoted               bool   `json:"exclude_quoted"`
	ExcludeSmallMatchesType     string `json:"exclude_small_matches_type"`
	ExcludeSmallMatchesValue    int    `json:"exclude_small_matches_value"`
}

// RubricCriteria has the rubric information for an assignment.
type RubricCriteria struct {
	Points            float64 `json:"points"`
	ID                string  `json:"id"`
	LearningOutcomeID string  `json:"learning_outcome_id"`
	VendorGUID        string  `json:"vendor_guid"`
	Description       string  `json:"description"`
	LongDescription   string  `json:"long_description"`
	CriterionUseRange bool    `json:"criterion_use_range"`
	// Ratings           []interface{} `json:"ratings"`
	Ratings []struct {
		ID              string  `json:"id"`
		Description     string  `json:"description"`
		LongDescription string  `json:"long_description"`
		Points          float64 `json:"points"`
	} `json:"ratings"`
	IgnoreForScoring bool `json:"ignore_for_scoring"`
}

// LockInfo is a struct containing assignment lock status.
type LockInfo struct {
	AssetString    string `json:"asset_string"`
	UnlockAt       string `json:"unlock_at"`
	LockAt         string `json:"lock_at"`
	ContextModule  string `json:"context_module"`
	ManuallyLocked bool   `json:"manually_locked"`
}

// Activity returns a course's activity data
func (c *Course) Activity() (interface{}, error) {
	var res interface{}
	return res, getjson(c.client, &res, nil, "/courses/%d/analytics/activity", c.ID)
}

// Files returns a channel of all the course's files
func (c *Course) Files(opts ...Option) <-chan *File {
	ch := make(fileChan)
	pager := c.filespager(ch, opts)
	go handleErrs(pager.start(), ch, c.errorHandler)
	return ch
}

// File will get a specific file id.
func (c *Course) File(id int, opts ...Option) (*File, error) {
	f := &File{client: c.client}
	return f, getjson(
		c.client, f, asParams(opts),
		"courses/%d/files/%d", c.ID, id,
	)
}

// ListFiles returns a slice of files for the course.
func (c *Course) ListFiles(opts ...Option) ([]*File, error) {
	ch := make(chan *File)
	p := c.filespager(ch, opts)
	files := make([]*File, 0)
	p.start()
	for {
		select {
		case file := <-ch:
			files = append(files, file)
		case err := <-p.errs:
			close(ch)
			return files, err
		}
	}
}

// Folders will retrieve the course's folders.
func (c *Course) Folders(opts ...Option) <-chan *Folder {
	ch := make(folderChan)
	pager := c.folderspager(ch, opts)
	go handleErrs(pager.start(), ch, c.errorHandler)
	return ch
}

// Folder will the a folder from the course given a folder id.
func (c *Course) Folder(id int, opts ...Option) (*Folder, error) {
	f := &Folder{client: c.client}
	path := fmt.Sprintf("courses/%d/folders/%d", c.ID, id)
	return f, getjson(c.client, f, asParams(opts), path)
}

// ListFolders returns a slice of folders for the course.
func (c *Course) ListFolders(opts ...Option) ([]*Folder, error) {
	ch := make(chan *Folder)
	p := c.folderspager(ch, opts)
	folders := make([]*Folder, 0)
	p.start()
	for {
		select {
		case folder := <-ch:
			folders = append(folders, folder)
		case err := <-p.errs:
			close(ch)
			return folders, err
		}
	}
}

// SetErrorHandler will set a error handling callback that is
// used to handle errors in goroutines. The default error handler
// will simply panic.
//
// The callback should accept an error and a quit channel.
// If a value is sent on the quit channel, whatever secsion of
// code is receiving the channel will end gracefully.
func (c *Course) SetErrorHandler(f func(error)) {
	c.errorHandler = f
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

// Quizzes will get all the course quizzes
func (c *Course) Quizzes(opts ...Option) ([]*Quiz, error) {
	return getQuizzes(c.client, c.ID, opts)
}

// Quiz will return a quiz given a quiz id.
func (c *Course) Quiz(id int, opts ...Option) (*Quiz, error) {
	return getQuiz(c.client, c.ID, id, opts)
}

func getQuizzes(client doer, courseID int, opts []Option) ([]*Quiz, error) {
	q := make([]*Quiz, 0)
	err := getjson(
		client, &q,
		asParams(opts),
		"courses/%d/quizzes", courseID,
	)
	return q, err
}

func getQuiz(client doer, course, quiz int, opts []Option) (*Quiz, error) {
	q := &Quiz{}
	err := getjson(
		client, q, asParams(opts), "courses/%d/quizzes/%d", course, quiz)
	return q, err
}

// Quiz is a quiz json response.
type Quiz struct {
	ID       int       `json:"id"`
	Title    string    `json:"title"`
	DueAt    string    `json:"due_at"`
	LockAt   time.Time `json:"lock_at"`
	UnlockAt string    `json:"unlock_at"`

	HTMLURL                       string          `json:"html_url"`
	MobileURL                     string          `json:"mobile_url"`
	PreviewURL                    string          `json:"preview_url"`
	Description                   string          `json:"description"`
	QuizType                      string          `json:"quiz_type"`
	AssignmentGroupID             int             `json:"assignment_group_id"`
	TimeLimit                     int             `json:"time_limit"`
	ShuffleAnswers                bool            `json:"shuffle_answers"`
	HideResults                   string          `json:"hide_results"`
	ShowCorrectAnswers            bool            `json:"show_correct_answers"`
	ShowCorrectAnswersLastAttempt bool            `json:"show_correct_answers_last_attempt"`
	ShowCorrectAnswersAt          time.Time       `json:"show_correct_answers_at"`
	HideCorrectAnswersAt          time.Time       `json:"hide_correct_answers_at"`
	OneTimeResults                bool            `json:"one_time_results"`
	ScoringPolicy                 string          `json:"scoring_policy"`
	AllowedAttempts               int             `json:"allowed_attempts"`
	OneQuestionAtATime            bool            `json:"one_question_at_a_time"`
	QuestionCount                 int             `json:"question_count"`
	PointsPossible                int             `json:"points_possible"`
	CantGoBack                    bool            `json:"cant_go_back"`
	AccessCode                    string          `json:"access_code"`
	IPFilter                      string          `json:"ip_filter"`
	Published                     bool            `json:"published"`
	Unpublishable                 bool            `json:"unpublishable"`
	LockedForUser                 bool            `json:"locked_for_user"`
	LockInfo                      interface{}     `json:"lock_info"`
	LockExplanation               string          `json:"lock_explanation"`
	SpeedgraderURL                string          `json:"speedgrader_url"`
	QuizExtensionsURL             string          `json:"quiz_extensions_url"`
	Permissions                   QuizPermissions `json:"permissions"`
	AllDates                      []string        `json:"all_dates"`
	VersionNumber                 int             `json:"version_number"`
	QuestionTypes                 []string        `json:"question_types"`
	AnonymousSubmissions          bool            `json:"anonymous_submissions"`
}

// QuizPermissions is the permissions for a quiz.
type QuizPermissions struct {
	Read           bool `json:"read"`
	Submit         bool `json:"submit"`
	Create         bool `json:"create"`
	Manage         bool `json:"manage"`
	ReadStatistics bool `json:"read_statistics"`
	ReviewGrades   bool `json:"review_grades"`
	Update         bool `json:"update"`
}

func (c *Course) filespager(ch chan *File, params []Option) *paginated {
	return newPaginatedList(
		c.client,
		fmt.Sprintf("courses/%d/files", c.ID),
		sendFilesFunc(c.client, ch),
		params,
	)
}

func (c *Course) folderspager(ch chan *Folder, params []Option) *paginated {
	return newPaginatedList(
		c.client,
		fmt.Sprintf("courses/%d/folders", c.ID),
		sendFoldersFunc(c.client, ch),
		params,
	)
}

func (c *Course) assignmentspager(ch chan *Assignment, params []Option) *paginated {
	return newPaginatedList(
		c.client, fmt.Sprintf("/courses/%d/assignments", c.ID),
		func(r io.Reader) error {
			asses := make([]*Assignment, 0, 10)
			err := json.NewDecoder(r).Decode(&asses)
			if err != nil {
				return err
			}
			for _, a := range asses {
				ch <- a
			}
			return nil
		}, params,
	)
}

func (c *Course) collectUsers(path string, opts []Option) (users []*User, err error) {
	ch := make(chan *User)
	errs := newPaginatedList(
		c.client, fmt.Sprintf(path, c.ID),
		sendUserFunc(c.client, ch), opts,
	).start()
	for {
		select {
		case u := <-ch:
			users = append(users, u)
		case err := <-errs:
			return users, err
		}
	}
}

func sendFilesFunc(d doer, ch chan *File) func(io.Reader) error {
	return func(r io.Reader) error {
		files := make([]*File, 0)
		err := json.NewDecoder(r).Decode(&files)
		if err != nil {
			return err
		}
		for _, f := range files {
			f.client = d
			ch <- f
		}
		return nil
	}
}

func sendFoldersFunc(d doer, ch chan *Folder) sendFunc {
	return func(r io.Reader) error {
		folders := make([]*Folder, 0)
		err := json.NewDecoder(r).Decode(&folders)
		if err != nil {
			return err
		}
		for _, f := range folders {
			f.client = d
			ch <- f
		}
		return nil
	}
}

func sendUserFunc(d doer, ch chan *User) sendFunc {
	return func(r io.Reader) error {
		list := make([]*User, 0)
		err := json.NewDecoder(r).Decode(&list)
		if err != nil {
			return err
		}
		for _, u := range list {
			u.client = d
			ch <- u
		}
		return nil
	}
}

func defaultErrorHandler(err error) {
	panic(err)
}

type assignmentChan chan *Assignment

func (ac assignmentChan) Close() {
	close(ac)
}
