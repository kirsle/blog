package comments

import "strings"

// ListDBName is the path to the singleton mailing list manager.
const ListDBName = "comments/mailing-list"

// MailingList manages subscription data for all comment threads.
type MailingList struct {
	Threads map[string]Subscription
}

// Subscription is the data for a single thread's subscribers.
type Subscription struct {
	Emails map[string]bool
}

// LoadMailingList loads the mailing list, or initializes it if it doesn't exist.
func LoadMailingList() *MailingList {
	m := &MailingList{
		Threads: map[string]Subscription{},
	}
	DB.Get(ListDBName, &m)
	return m
}

// Subscribe to a comment thread.
func (m *MailingList) Subscribe(thread, email string) error {
	email = strings.ToLower(email)
	t := m.initThread(thread)
	t.Emails[email] = true
	return DB.Commit(ListDBName, &m)
}

// List the subscribers for a thread.
func (m *MailingList) List(thread string) []string {
	t := m.initThread(thread)
	result := []string{}
	for email := range t.Emails {
		result = append(result, email)
	}
	return result
}

// Unsubscribe from a comment thread. Returns true if the removal was
// successful; false indicates the email was not subscribed.
func (m *MailingList) Unsubscribe(thread, email string) bool {
	email = strings.ToLower(email)
	t := m.initThread(thread)
	if _, ok := t.Emails[email]; ok {
		delete(t.Emails, email)
		DB.Commit(ListDBName, &m)
		return true
	}
	return false
}

// UnsubscribeAll removes the email from all mailing lists.
func (m *MailingList) UnsubscribeAll(email string) bool {
	var any bool
	email = strings.ToLower(email)
	for thread := range m.Threads {
		if m.Unsubscribe(thread, email) {
			any = true
		}
	}

	return any
}

// initialize a thread structure.
func (m *MailingList) initThread(thread string) Subscription {
	if _, ok := m.Threads[thread]; !ok {
		m.Threads[thread] = Subscription{
			Emails: map[string]bool{},
		}
	}
	return m.Threads[thread]
}
