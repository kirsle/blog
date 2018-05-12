/*
Package comments implements the controllers for the commenting system.

Routes

	/comments               Main comment handler
	/comments/subscription  Manage subscription to comment threads
	/comments/quick-delete  Quickly delete spam comments from admin email

Related Models

	comments

Description

Comments are a generic comment thread system that can be placed on any page.
They are automatically attached to blog posts (unless you disable comments on
them) but they can be used anywhere. A guestbook, on the events pages, on any
custom pages, etc.

Every comment thread has a unique ID, so some automated threads have name spaces,
like "blog-$id".

Subscriptions

When users leave a comment with their e-mail address, they may opt in to getting
notified about future comments left on the same thread.

Go Template Function

You can create a comment form on a page in Go templates like this:

	func RenderComments(r *http.Request, subject string, ids ...string) template.HTML
	{{ RenderComments .Request "Title" "id part" "id part" "id part..." }}

The subject is used in the notification e-mail. The ID strings are joined together
by dashes and you can have as many as you need. Examples:

	Blog posts in the format `blog-<postID>` like `blog-42`
	{{ RenderComments .Request .Data.Title "blog" .Data.IDString }}

	Events in the format `event-<eventID>` like `event-2`
	{{ RenderComments .Request "My Big Party" "event" "2" }}

	Custom ID for a guestbook
	{{ RenderComments .Request "Guestbook" "guestbook" }}
*/
package comments
