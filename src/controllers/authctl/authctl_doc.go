/*
Package authctl implements the authentication controllers.

Routes

	/login       Log in to a user account
	/logout      Log out
	/account     Account home page (to edit profile or reset password)
	/age-verify  If the blog implements age gating

Related Models

	users

Age Gating

If the blog marks itself as NSFW, visitors to the blog must verify their age
to enter. The middleware that controls this is at `middleware/age-gate.go`.
The controller is here at `controllers/authctl/gate-gate.go`
*/
package authctl
