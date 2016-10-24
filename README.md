## Go Web Crawler

Known Issues
=====
 * Links to PDFs?
 * Remove redundant SameOriginChecking in utils.go -> write a function

Pseudocode
====
````
Pseudocode
Get url
Fetch the page at the url
Parse that page for its links
Put url, fetched page, and links into Webpage data structure
For each link:
    Check if url already in Webpage map, if not...
        Spawn / grab a thread out of a pool
        Make a request to the url
        Parse links
        Add to Webpage map
    If uri is in map,
        Don't do anything
````
