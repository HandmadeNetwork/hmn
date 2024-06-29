# Styling TODO

- [ ] Fix spacing of podcast episodes (used to use p-spaced)
- [x] Audit uses of tables across the site to see where we might actually need to apply table-layout: fixed and border-collapse: collapse
    - There are zero tables on the site except one used for Asaf's debugging.
- [ ] Recover <hr> styles and use them only in post-content
- [ ] Audit and fix all uses of:
    - [ ] big
    - [ ] title
    - [x] clear
    - [x] full
    - [ ] hidden
    - [x] empty (?!)
    - [x] column
    - [ ] content
    - [ ] description
    - [ ] c--dim and friends
- [ ] Re-evaluate form styles
    - [ ] theme-color-light is used only for buttons
- [x] center-layout vs. margin-center
- [ ] Make sure old projects look ok (background images are gone)
- [ ] Audit or delete whenisit
- optionbar fixes
    - [ ] Remove "external" styles (width, padding, etc.)
    - [ ] Fix options (no more "buttons")
    - [ ] Convert all buttons to links
    - [ ] Generally audit visuals
    - [ ] Find that thing and kill it?
- [ ] Probably remove uses of .tab
- [ ] everything in _content.scss, ugh
- [ ] Reduce saturation of --background-even-background
- [ ] Update blog styles to not use `post` and other garbage
    - [ ] Remove from forum.css
- [ ] Remove all uses of .content-block
- [ ] Figure out what's up with the projects on the jam pages
- [ ] Determine if we actually need .project-carousel, or if our general carousel styles (?) are sufficient
- [ ] Rename `-2024` files
- [ ] Validate accessibility of navigation
- [ ] Make navigation work on mobile
- [ ] Clean up code TODOs
- [ ] Support the following external logos:
    - Twitter / X
    - Patreon
    - Discord
    - Twitch
    - Steam
    - Itch?
    - Generic website
    - Bluesky
    - YouTube
    - Vimeo
    - App Store
    - Play Store
    - GitHub
    - Threads?
    - TikTok?
    - Trello?
- [ ] Handle empty avatar URLs correctly in various places (render as theme-dependent default)
- [x] Convert to new color vars
- [ ] Make snippet descriptions partially collapse by default
- [ ] Make the home page remember which tab you were on
- [ ] Resolve TODO(redesign) comments

stack!

timeline-media-background (see if an existing color works)
red (c-red)
projectcarddata
