/*
Package events provides controllers for the event system.

Routes

	Admin Only
	/e/admin/edit               Edit an event
	/e/admin/invite/<event_id>  Manage invitations and contacts to an event
	/e/admin/                   Event admin index

	Public
	/e/<event_fragment>         Public URL for event page
	/c/logout                   Logout authenticated contact
	/c/<user_secret>            Authenticate a contact

Related Models

	contacts
	events

Description

Events provide basic features of event planning software, including
(for the admin user of the site):

	* View, edit, delete events.
	* Events have a title, datetime range, Markdown body, etc.
	* You can invite users (contacts) to the event.

Contacts are their own distinct entity in the database, separate from Users
(which are the website admin user accounts, with passwords).

When you invite people to an event, you create new Contact entries for the
people who don't have them yet or invite the ones who exist. Each Contact
uniquely groups a first and last name, e-mail address and SMS number.

When you send out invite e-mail or SMS messages, each Contact is given their own
personal link to view the event details. The link goes to the URL
`/c/<user_secret>?e=<event_id>`, where "user_secret" is a secret random string
generated on their Contact object (to identify the Contact) and "event_id" is
the ID number of the event.

The Contact Authenticator endpoint at `/c/<user_secret>` "authenticates" them
in their browser session by setting the session key "contact.id" -- this is only
of any interest to the Events controller anyway.

Events, Contacts, and RSVPs

There is an Event row for every distinct event, and a single Contact row for
every distinct person.

RSVP's are how we marry Events to their invited Contacts, *and* how we track
the RSVP response of each contact.

When a user clicks the link in their invite e-mail, their browser authenticates
as their Contact (it greets them by their name and shows the buttons to respond
to the event). When they click "Going" or "Not Going", the server knows which
contact they are and can find them on the RSVP list, and mark their status
accordingly.

Comment Form

Events have comment forms using the thread format "event-<id>", like "event-1"
for the first event. When an authenticated Contact (one who clicked an email
link) interacts with the Response Form, we auto-subscribe their e-mail to the
comment form. This way anybody leaving comments on the page will naturally
notify the users who have 1) awareness of the event, 2) have given an answer
about it.

They can easily unsubscribe from the comment thread as normal for the blog's
commenting system.
*/
package events
