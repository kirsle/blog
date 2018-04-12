#!/usr/bin/env python3

"""
Import a Tumblr blog.

This is written in Python because the Tumblr JSON is too hairy for Go (dynamic
keys and such).

Usage: tumblr-import.py --root /path/to/blog/root example.tumblr.com

It initializes your blog settings with the metadata (title/description), then
starts crawling all of your blog posts from oldest to newest, creating blog
posts in the new Blog CMS. This way post ID #1 matches with your first Tumblr
post!

All images referenced in photo posts are downloaded to ``<root>/static/photos/``
with their original ``tumblr_XXXXXXX.jpg`` file names. The blog entries simply
list these images with ``<img>`` tags.
"""

import argparse
from datetime import datetime
import json
import logging
import os
import re
import requests

logging.basicConfig()
log = logging.getLogger("*")
log.setLevel(logging.INFO)

def main(tumblr, root, scheme):
    log.info("Starting tumblr-import from blog: %s", tumblr)
    API_ROOT = "{}://{}/api/read/json".format(scheme, tumblr)
    log.info("Root Tumblr API URL: %s", API_ROOT)

    try:
        r = api_get(API_ROOT)
    except Exception as e:
        log.info("Failed to get Tumblr API root at %s: %s", API_ROOT, e)
        exit(1)

    # Ingest the site settings from the first page of results.
    SiteSettings = {
        "site": {
            "title": r["tumblelog"]["title"],
            "description": r["tumblelog"]["description"],
        },
    }
    log.info("Blog title: %s", SiteSettings["site"]["title"])
    log.info("Description: %s", SiteSettings["site"]["description"])
    # write_db(root, "app/settings", SiteSettings)

    posts_total = r["posts-total"]  # n
    log.info("Total Posts: %d", posts_total)

    # Go to the last page and work backwards.
    per_page    = 50
    posts_start = posts_total - per_page
    if posts_start < 0:
        posts_start = 0

    POST_ID = 0

    # Unique Tumblr post IDs so no duplicates.
    tumblr_posts = set()

    while posts_start >= 0:
        log.info("GET PAGE start=%d  num=%d\n", posts_start, per_page)
        r = api_get(API_ROOT, start=posts_start, num=per_page)

        with open("tumblr-import.json", "w") as fh:
            fh.write(json.dumps(r, indent=4))

        # Ingest the posts in reverse order.
        for post in r["posts"][::-1]:
            log.info("{id}: {type}   [{date_gmt}]".format(
                id=post["id"],
                type=post["type"],
                date_gmt=post["date-gmt"],
            ))
            timestamp = datetime.fromtimestamp(post["unix-timestamp"])
            timestamp = timestamp.strftime("%Y-%m-%dT%H:%M:%SZ")

            if post["id"] in tumblr_posts:
                continue
            tumblr_posts.add(post["id"])

            slug = post.get("slug")
            if not slug:
                slug = re.sub(r'.*{}/'.format(tumblr), '', post.get("url-with-slug"))

            # Post model to build up from this.
            POST_ID += 1
            Post = dict(
                id=POST_ID,
                title="",
                fragment=slug,
                contentType="html",
                author=1,
                body="",
                privacy="public",
                sticky=False,
                enableComments=True,
                tags=post.get("tags", []),
                created=timestamp,
                updated=timestamp,
            )

            # Photo posts
            if post["type"] == "photo":
                # Search it for the best quality image.
                images = _download_images(root, _all_photos(post))
                img_tags = "\n".join([ '<p><img src="{}"></p>\n'.format(i) for i in images ])

                Post["body"] = (
                    '{}\n\n{}'
                    .format(img_tags, post.get("photo-caption"))
                    .strip()
                )
            elif post["type"] == "answer":
                Post["body"] = (
                    '<blockquote><strong>Anonymous</strong> asked:\n'
                    '<p>{question}</p>\n'
                    '</blockquote>\n\n{answer}'
                    .format(question=post.get("question"), answer=post.get("answer"))
                )
            elif post["type"] == "regular":
                Post["title"] = post["regular-title"]
                Post["body"] = post["regular-body"]
            elif post["type"] == "video":
                log.error("No support for video posts!")
            else:
                print("UNKNOWN TYPE OF POST!")
                print(json.dumps(post, indent=2))
                input()

            # Add an import notice suffix.
            Post["body"] += (
                "\n\n"
                '<span class="text-muted">Imported from Tumblr where it had {} notes.</span>'
                .format(post.get("note-count"))
            )

            write_db(root, "blog/posts/{}".format(Post["id"]), Post)
            write_db(root, "blog/fragments/{}".format(Post["fragment"]), dict(
                id=Post["id"],
            ))

        if posts_start == 0:
            break

        posts_start -= per_page
        if posts_start < 0:
            posts_start = 0


def write_db(root, path, data):
    """
    Write JSON data to a DB path in the Blog DB.

    Parameters:
        path (str): DB path like `blog/posts/123`
        data (dict): dict of JSON serializable data to write.
    """
    dirs, filename = path.rsplit("/", 1)
    dirs = os.path.join(root, ".private", dirs)
    if not os.path.isdir(dirs):
        log.info("CREATE DIRECTORY: %s", dirs)
        os.makedirs(dirs)

    with open(os.path.join(dirs, filename+".json"), "w") as fh:
        fh.write(json.dumps(data, indent=4))


def api_get(root, start=0, num=20):
    url = root + "?start={}&num={}".format(start, num)
    log.info("API GET: %s", url)
    r = requests.get(url)
    if r.ok:
        body = r.content.decode()
        m = re.match(r'var tumblr_api_read = (.+);', body)
        if m:
            return json.loads(m.group(1))
    raise Exception("didn't get expected JSON wrapped response")

def _all_photos(data):
    """Extract all photos from a photo post."""
    main_photo = _best_resolution(data)
    result = [ main_photo ]
    if "photos" in data:
        for photo in data["photos"]:
            best = _best_resolution(photo)
            result.append(best)
    return result

def _best_resolution(data):
    """Find the best image resolution from post json."""
    highest = 0
    best = ""
    for k, v in data.items():
        m = re.match(r'^photo-url-(\d+)$', k)
        if m:
            width = int(m.group(1))
            if width > highest:
                highest = width
                best = v
    return best

def _download_images(root, images):
    """Download an array of images. Return array of file paths."""
    result = []
    if not os.path.isdir(os.path.join(root, "static/photos")):
        os.makedirs(os.path.join(root, "static/photos"))

    for i, url in enumerate(images):
        log.info("DOWNLOAD: %s", url)

        filename = url.rsplit("/", 1)[-1]
        filename = "static/photos/"+filename
        if os.path.isfile(os.path.join(root, filename)):
            result.append("/"+filename)
            continue

        r = requests.get(url)
        if r.ok:
            with open(os.path.join(root, filename), "wb") as fh:
                fh.write(r.content)
            result.append("/"+filename)
        else:
            result.append(url)
    return result

if __name__ == "__main__":
    parser = argparse.ArgumentParser("Tumblr importer")
    parser.add_argument("--root",
        help="Output website root for the blog.",
        type=str,
        required=True,
    )
    parser.add_argument("--nossl",
        help="Don't use SSL when connecting to the tumblr site",
        action="store_true",
    )
    parser.add_argument("tumblr",
        help="Tumblr blog name, in username.tumblr.com format.",
    )
    args = parser.parse_args()
    main(
        tumblr=args.tumblr,
        root=args.root,
        scheme="http" if args.nossl else "https",
    )
