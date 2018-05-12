/*
Package postctl implements all the web blog features.

Routes

	Public
	/blog             Blog index
	/blog.rss         RSS feed
	/blog.atom        Atom feed
	/archive          Blog archives
	/tagged           Index of all blog tags
	/tagged/<tag>     View posts by tag
	/<fragment>       View blog entry by its URL fragment

	Admin Only
	/blog/edit        Create or edit blog post
	/blog/delete      Confirm deletion of blog post
	/blog/drafts      View all draft entries
	/blog/private     View all private entries

Related Models

	posts

Description

Each post is in its own JsonDB document at `posts/entries/<id>.json` and
contains all its data (title, body, tags, timestamps, etc.)

For faster retrieval and caching of overall post data, there is a Blog Index
that gets saved in JsonDB at `posts/index.json`. The index summarizes ALL of
the blog posts by caching their basic details (ID, URL fragment, title,
tags, created time). This document is used for getting a narrower list of posts
to work with, for index pages (with pagination), "by tagged" pages, etc.

Usually the front-end settles on 5 or 10 posts it wants to render, and it only
had to look at the index. For the archive view where it only needs the blog
titles, it already has these too. For the posts where it needs the full body,
it has the IDs and can just select each one pretty quickly.

In case anything goes wrong with the blog index, you can always delete the
`posts/index.json` and it will be re-generated from scratch in a one-time scan
of the entire posts DB (opening every document).
*/
package postctl
