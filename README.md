# How to index your site using this project

1. Generate UUID using `uuid` Linux command. Further refered to as `<key>`
   - On MacOS and Windows, use `uuidgen`
2. Save The `<key>` into a file called `<key>.txt`. Name and the contents of this file must be the identical `<key>` value.
3. Upload the `<key>.txt` file onto your website so that it's accesible from `https://<domain>/<key>.txt`.
4. Run `go run main.go -k <key> -s https://<domain>/sitemap.xml`. This will send a indexnow request to `IndexNow`, `Microsoft Bing`, `Naver`, `Seznam.cz`, `Yandex` and `Yep`
   - unfortunatelly, Google does not support indexnow protocol, so you have to setup indexing manually using [Google Search Console](https://search.google.com/search-console/about)
5. You're done and your site will get indexed :D
