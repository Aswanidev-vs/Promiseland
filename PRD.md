# PromisedLand — Product Requirements Document (PRD)

> **Working title:** PromisedLand  
> **Document status:** Draft v1.1  
> **Last updated:** 23 November 2026
> **Revision note:** v1.1 updates the working product name and brand positioning from YoriVerse to PromisedLand; product scope and technical direction remain unchanged.  
> **Product type:** Anime-first social network and community platform  
> **Initial platform:** Responsive web application  
> **Primary stack:** Vue 3 + TypeScript, Go, SQLite, Docker  
> **Initial hosting target:** AWS EC2  
> **Naming note:** “PromisedLand” is the current working brand name. Trademark, company-name, social-handle, and domain availability have not been cleared; complete appropriate checks before public launch.

---

## 1. Executive Summary

PromisedLand is an anime-first social platform designed to bring anime fans, manga readers, cosplayers, fan artists, event organizers, and local fandom communities together in one place.

The product combines four complementary experiences:

1. **Forum and community discussions:** Reddit-like communities, posts, comments, voting, polls, and spoiler-aware discussions.
2. **Social discovery and creator feed:** Instagram-like photo and video posts, cosplay and crossplay showcases, fan art, event memories, follows, and reactions.
3. **Community communication:** Discord-like servers, topic channels, group conversations, and direct messages.
4. **Events and real-world connections:** Anime conventions, cosplay meetups, screenings, creator collaborations, and local community gatherings.

A later, strictly optional area may help adult users discover friends or romantic connections based on shared interests. Dating is not part of the initial release and must be separated from the general community experience, limited to adults, and launched only after appropriate age assurance, privacy, safety, reporting, and moderation capabilities exist.

The central product idea is an **anime interest graph**: a user's anime preferences, communities, content, events, and connections should reinforce one another. For example, a user who follows a series and enjoys cosplay should be able to discover its discussion community, find relevant creators, see related events, and connect with other fans who share those interests.

PromisedLand should feel like a destination and welcoming home for anime fandom: a place to belong, discover people, share creative work, and turn shared interests into real-world connections. The brand should communicate belonging and community without making the product look like a dating app or a streaming catalogue.

## 2. Product Vision and Positioning

### 2.1 Vision

Build a welcoming online home where anime fans can discover communities, discuss stories, share creative work, organize real-world activities, and form meaningful connections around shared interests.

### 2.2 Product promise

**Your fandom. Your people. Your place.**

### 2.3 Positioning

PromisedLand is an anime-first fandom social network. It is not intended to replace every feature of Reddit, Instagram, Discord, or dating applications. Instead, it connects the most relevant parts of those experiences around anime culture and fan activities. Its brand promise is “Your fandom. Your people. Your place.”

### 2.4 Differentiation

The product should differentiate through the connections between features:

- Anime interests personalize communities, posts, creators, and event discovery.
- Posts and discussions can be tagged to an anime, season, episode, character, or event.
- Spoiler controls respect each user's stated viewing progress where reliable episode metadata is available.
- Online fandom can lead to real-world participation through events, cosplay teams, and meetups.
- Friend discovery is based on shared fandom interests and is separate from optional adult dating.

### 2.5 Brand direction

- **Working name:** PromisedLand. Use the single-word brand spelling consistently in the product, repository, and documentation.
- **Tagline:** “Your fandom. Your people. Your place.”
- **Brand idea:** a welcoming destination where anime fans find communities, share their interests, create together, attend events, and build meaningful connections.
- **Tone:** welcoming, expressive, inclusive, community-minded, and trustworthy. Avoid making the product feel exclusively romantic, overly childish, or like an anime streaming service.
- **Naming caveat:** the name is a working choice only. Complete trademark, company-name, domain, and social-handle checks before investing in public launch branding.

### 2.6 Product principles

1. **Community before virality:** prioritize useful discussions and healthy communities over engagement-maximizing mechanics.
2. **Spoiler respect by default:** make spoiler status visible and provide controls to hide or reveal sensitive content.
3. **Creative expression:** make sharing cosplay, crossplay, fan art, photography, and event experiences straightforward.
4. **User control:** let users control profile visibility, messages, follows, notifications, and discovery preferences.
5. **Safety is a product feature:** build reporting, blocking, moderation, and event safety into the platform rather than treating them as later polish.
6. **Progressive scope:** launch a coherent MVP first; add real-time communication and dating only after the foundations are stable.
7. **Responsive and accessible:** support mobile, tablet, and desktop web layouts, keyboard access, and readable contrast.

## 3. Goals, Non-Goals, and Success Criteria

### 3.1 Product goals

- Give anime fans a single place to participate in fandom communities and discussions.
- Let users publish and discover anime-related social content, including cosplay and crossplay.
- Help users discover events and coordinate attendance or collaborations.
- Provide a foundation for real-time community communication.
- Build a maintainable application suitable for a public portfolio and future product growth.
- Demonstrate practical backend, container, cloud, deployment, and operations skills.

### 3.2 Engineering and learning goals

The project should provide hands-on experience with:

- Vue 3, TypeScript, component architecture, routing, and client-side state.
- Go HTTP APIs, authentication, authorization, validation, and modular monolith design.
- Relational data modelling, SQLite migrations, transactions, indexing, and query design.
- Dockerfiles, multi-stage builds, Docker Compose, container networking, health checks, and persistent storage.
- AWS EC2, EBS, IAM, VPC/security groups, ECR, S3, and CloudWatch.
- GitHub Actions CI/CD, automated tests, image publishing, and deployment automation.
- Production concerns including HTTPS, secrets, backups, logging, monitoring, and incident recovery.

### 3.3 Non-goals for the first release

The first release will not attempt to:

- Stream anime episodes or host copyrighted anime video.
- Replace full-featured Discord voice/video infrastructure.
- Offer live audio rooms, video calls, or screen sharing.
- Build an advanced recommendation engine or machine-learning ranking system.
- Launch dating or romantic matching.
- Build a native iOS or Android application.
- Provide a full anime catalogue or tracking service competing feature-for-feature with established trackers.
- Support multiple backend instances while using a single SQLite database file.
- Implement end-to-end encrypted messaging in the initial messaging release.
- Guarantee the legality of user-uploaded content; users must have rights to the content they submit and the platform must provide reporting and takedown workflows.

### 3.4 Success criteria

Success should be measured by a combination of product usage, community health, and engineering reliability. Initial targets are hypotheses and should be revised after a closed beta.

**Closed-beta product targets:**

- 50–100 invited users create accounts.
- At least 5 active communities are created and moderated.
- At least 30% of registered users return within 7 days.
- At least 25% of registered users make a meaningful contribution (post, comment, event RSVP, or content share) within 14 days.
- At least 10 useful, legitimate anime or cosplay posts are published per week during the beta.
- User reports are reviewed within 48 hours during the beta.

**Engineering targets:**

- CI runs formatting, linting, unit tests, and frontend build checks on every pull request.
- Docker Compose starts the application reliably from a clean checkout using documented configuration.
- Application and database data survive backend container recreation and EC2 reboot.
- HTTPS is enabled for the public deployment.
- A documented backup and restore procedure is tested before public launch.
- Health checks, application logs, and basic resource monitoring are available.
- No production secrets are committed to source control.

These targets are not guarantees of business success. They are initial product and operational acceptance measures.

## 4. Target Users and Personas

### 4.1 Anime fan / community participant

- Watches anime or reads manga.
- Wants to discuss episodes, theories, characters, and recommendations.
- Wants to avoid spoilers for episodes not yet watched.
- Joins communities and follows people with similar interests.

**Primary needs:** relevant discussions, good discovery, spoiler controls, and a welcoming profile.

### 4.2 Cosplayer / crossplayer / creator

- Creates cosplay, crossplay, fan art, edits, photography, or other fandom content.
- Wants to showcase work and find collaborators.
- Wants to discover conventions, shoots, and local events.

**Primary needs:** media posts, portfolios, tags, creator discovery, collaboration requests, and event visibility.

### 4.3 Community organizer / moderator

- Runs an anime club, online community, college group, or local fandom group.
- Needs to publish announcements, moderate conversations, and organize events.

**Primary needs:** community settings, roles, rules, moderation queue, announcements, event tools, and member management.

### 4.4 Event organizer

- Organizes conventions, meetups, screenings, cosplay competitions, or photography gatherings.
- Needs to publish event information and communicate with attendees.

**Primary needs:** event pages, location and time details, RSVP management, updates, and attendee coordination.

### 4.5 Adult user seeking friendship or dating (future phase)

- Wants to meet people based on shared anime interests.
- May be interested in friendship, dating, or both.
- Needs strong privacy, consent, eligibility checks, and safety controls.

**Primary needs:** opt-in discovery, preference controls, compatibility explanation, mutual matching, and safe communication.

This persona is explicitly excluded from the initial release. The platform must not infer romantic intent from ordinary profile interests or community activity.

## 5. User Roles and Permissions

The platform should use role-based access control with resource ownership checks.

| Role | Description | Key permissions |
|---|---|---|
| Visitor | Not signed in | View public pages and public posts where permitted; cannot interact |
| Member | Registered user | Create profile, join communities, post, comment, react, follow users, RSVP |
| Community moderator | Moderator within a community | Review reports, remove community content, manage community participation, apply community-level restrictions |
| Community owner | Creator or designated owner | Configure community, assign moderators, manage rules and membership |
| Event organizer | Owner of an event | Edit event, manage RSVP list, publish updates, moderate event discussion |
| Platform moderator | Trusted platform staff | Review platform-wide reports, suspend accounts, remove prohibited content, handle appeals |
| Platform administrator | Technical/platform administrator | Manage platform configuration and administrative operations |

A user may hold different roles in different communities. Community roles must not automatically grant platform-wide permissions.

## 6. Product Information Architecture

### 6.1 Primary navigation

The responsive web application should expose these main destinations:

- **Home:** personalized feed with tabs such as For You and Following.
- **Communities:** discover, join, and manage anime communities.
- **Events:** discover conventions, cosplay events, meetups, screenings, and online activities.
- **Chat:** servers, channels, group conversations, and direct messages (introduced in a later release).
- **Discover:** discover anime, users, creators, communities, and—only in a later adults-only phase—opt-in dating profiles.
- **Notifications:** replies, mentions, follows, event updates, and other relevant activity.
- **Profile:** user bio, avatar, anime interests, posts, saved items, and privacy settings.

On mobile widths, use a compact bottom navigation or an accessible collapsible navigation. Avoid placing every feature in a crowded sidebar.

### 6.2 Public and authenticated areas

Public pages may include landing pages, public communities, public posts, public event pages, and public profiles when the user has chosen public visibility. Actions such as posting, commenting, joining, RSVP, messaging, and following require authentication.

## 7. Functional Requirements

Priority definitions:

- **P0 — MVP required:** must be implemented before the first public beta.
- **P1 — Next release:** important, but can follow the initial beta.
- **P2 — Later:** valuable expansion after core product validation.
- **P3 — Future / optional:** requires additional safety, infrastructure, or product validation.

### 7.1 Authentication and account management

**Priority:** P0

Requirements:

- Users can register with email and password.
- Users can sign in and sign out.
- Users can request a password reset through a time-limited token.
- Email verification should be enabled before sensitive actions or public launch.
- Users can edit basic account settings and deactivate their account.
- The backend must enforce authentication and authorization for protected operations.
- Passwords must be hashed with a modern password-hashing algorithm; plaintext passwords must never be stored or logged.
- Sessions must use secure, HTTP-only cookies where practical, with appropriate SameSite and Secure settings in production. If token-based auth is selected, refresh-token rotation and revocation must be specified.
- Apply login and password-reset rate limits.

Acceptance criteria:

- Invalid credentials do not reveal whether an email address is registered.
- A user cannot access another user's private account data by changing an identifier in a request.
- Password-reset tokens expire and are single-use.
- Authenticated routes are protected on the server, not only hidden in the frontend.

### 7.2 User profiles and anime interests

**Priority:** P0

Profile fields:

- Username / handle (unique).
- Display name.
- Avatar.
- Short bio.
- Optional banner image.
- Country and optional city/region (user-controlled; private by default where appropriate).
- Favourite anime and manga.
- Favourite genres.
- Favourite characters (optional).
- Interests such as cosplay, crossplay, fan art, photography, manga, conventions, and gaming.
- Profile visibility: public or private.
- Links to external creator/social profiles (optional).

Requirements:

- Users can create and edit their profile.
- Users can add, remove, and reorder interests and favourite titles.
- Users can follow and unfollow public profiles.
- Users can view public profile content and follower/following counts subject to privacy settings.
- Users can block another account.
- Users can control whether their profile appears in discovery.

Acceptance criteria:

- Usernames are unique and validated.
- Private profile content is not returned to unauthorized users.
- Blocking prevents the blocked user from initiating new interactions with the blocker and hides relevant content according to documented policy.

### 7.3 Anime metadata and tagging

**Priority:** P0 (basic tagging); P1 (richer catalogue)

Requirements:

- Users can associate posts, communities, and events with anime or manga titles.
- The initial release may use a small seeded catalogue or a reliable public metadata API, subject to its terms and availability.
- Store an external catalogue identifier when one exists, but do not make the platform dependent on a single external provider.
- Support title, alternate title, format/type, and optional season/episode references where available.
- Allow users to submit a title suggestion if it is not in the catalogue; suggestions may require moderation.

Acceptance criteria:

- A post can reference one or more anime titles.
- Invalid or unknown episode references do not silently create misleading metadata.
- External catalogue failure does not make core discussions unavailable.

### 7.4 Communities and memberships

**Priority:** P0

A community is a topic-based space for discussion, such as a specific anime, manga, genre, creator interest, or local anime club.

Requirements:

- Authenticated users can create a community.
- Community fields include name, unique slug, description, icon/banner, topic tags, visibility, rules, and creator.
- Visibility options: public and private. Private communities require an invitation or approval.
- Users can join and leave public communities.
- Private communities support join requests or invite links.
- Community owners can assign moderators.
- Community owners can edit basic settings and rules.
- Community pages display posts, member count, description, and rules.
- Community owners can disable posting or restrict participation when needed.

Acceptance criteria:

- Only authorized members can view or participate in private communities.
- Only owners/moderators can change community settings or moderate community content.
- A user cannot grant themselves a role by modifying a client request.

### 7.5 Forum posts, comments, voting, and polls

**Priority:** P0

Post types:

- Text discussion.
- Link post.
- Image post.
- Question / help post.
- Poll.
- Announcement (community moderators only, where enabled).

Requirements:

- Members can create, edit, and delete their own posts subject to moderation and retention policy.
- Posts can be published to a community or as a profile/feed post.
- Posts support title, body, tags, optional anime/episode references, and spoiler status.
- Members can comment and reply to comments with a defined nesting limit for usability.
- Members can upvote/downvote or use a simpler reaction model. The initial implementation should select one consistent voting model and avoid exposing conflicting counts.
- Members can save/bookmark posts.
- Polls support a question and a bounded set of options, with one vote per eligible account unless configured otherwise.
- The interface displays edit status and timestamps.
- Moderators can lock comments or remove content with a reason.

Acceptance criteria:

- Users cannot vote multiple times on the same item where only one vote is allowed.
- Deleted content follows a documented soft-delete or removal policy.
- Comment permissions respect the parent post's community and privacy rules.
- Pagination is used; the application must not load an entire large community feed at once.

### 7.6 Spoiler protection and episode-aware discussion

**Priority:** P0 (manual spoiler controls); P1 (watch-progress-aware filtering)

Requirements:

- Users can mark a post or text segment as containing spoilers.
- Spoiler content is hidden or visually obscured until a user explicitly reveals it.
- Posts can optionally reference a title, season, and episode/chapter.
- Communities may define spoiler rules, including a required episode reference for episode discussions.
- In a later enhancement, users can record their watched/read progress and hide content beyond that progress.
- The system must make clear that user-supplied spoiler tags may be incorrect; users must have reporting controls.

Acceptance criteria:

- Spoiler-marked text is not visible by default in supported feed and discussion views.
- Spoiler metadata is returned by the API so the frontend is not solely responsible for deciding whether content is a spoiler.
- Users can report unmarked spoilers.

### 7.7 Social feed and media posts

**Priority:** P0 (basic image feed); P1 (video and richer creator tools)

Requirements:

- Users can publish image-based posts with captions.
- Posts can include one or more media attachments within configured file and count limits.
- Users can tag anime titles, characters, events, and relevant topics.
- Users can follow other users.
- Feed tabs include Following and For You. For the MVP, For You may be a simple deterministic feed based on followed communities, interests, and recency; do not require a machine-learning system.
- Users can like/react, comment, share a post link, and bookmark posts.
- Users can report content.
- Media should be stored in object storage, not inside the SQLite database.
- Video uploads should be deferred until storage, transcoding, moderation, bandwidth, and cost implications are understood.

Acceptance criteria:

- Upload type and size are validated on the server.
- Media is served through controlled public URLs or signed URLs according to visibility.
- Failed uploads do not create broken feed entries.
- Users cannot modify or delete another user's media without authorization.

### 7.8 Cosplay, crossplay, fan art, and creator profiles

**Priority:** P1

Requirements:

- Users can identify a post as cosplay, crossplay, fan art, photography, edit, collection, or other creator content.
- Creators can feature selected work on their profile.
- Users can add optional character and series tags.
- Users can link to an event or collaboration request.
- Users can report impersonation, harassment, stolen content, or inappropriate material.
- The platform must not assume a person's gender or identity from a cosplay character or presentation.

Acceptance criteria:

- Creator categories are discoverable through filters.
- Users can control which work is featured on their profile.

### 7.9 Events and local fandom activities

**Priority:** P1

Event types:

- Anime convention.
- Cosplay/crossplay meetup or competition.
- Screening or watch party.
- Photography shoot.
- Club meetup.
- Online community event.
- Creator workshop or collaboration.

Event fields:

- Title and description.
- Cover image.
- Organizer.
- Event type and relevant anime tags.
- Start and end date/time with timezone.
- Venue or online event details.
- City/region (optional).
- Capacity (optional).
- RSVP status and attendee count.
- Visibility and participation requirements.
- Event rules and contact method.

Requirements:

- Authenticated users can create an event.
- Users can discover events by date, type, anime tag, and broad location.
- Users can RSVP, cancel an RSVP, and view their upcoming events.
- Organizers can edit event details and publish updates.
- Organizers can manage attendee lists and capacity where enabled.
- Events can have a discussion thread or linked community.
- Exact home addresses and a user's live location must never be exposed by default.
- Public event pages should show only the venue details intentionally supplied by the organizer.

Acceptance criteria:

- Event time is stored consistently (UTC or explicit timezone-aware representation) and displayed in the user's chosen/local timezone.
- Capacity limits are enforced transactionally.
- Users can report suspicious or unsafe events.
- Users can withdraw an RSVP.

### 7.10 Collaboration / Team Up

**Priority:** P1

Requirements:

- Users can publish collaboration requests such as “looking for cosplayers”, “need a photographer”, or “forming a convention group”.
- Requests include type, description, relevant event/anime, preferred date, broad location, and number of collaborators needed.
- Interested users can request to join.
- The creator can accept or decline requests.
- Accepted members can be moved to an event or group conversation when the chat module is available.
- Users can close a request when filled or no longer active.

Acceptance criteria:

- Only the request creator can accept or decline join requests.
- Private contact details are not exposed before consent.

### 7.11 Notifications

**Priority:** P0 (in-app notifications); P1 (email/push extensions)

Notification types:

- New follower.
- Post/comment reply.
- Mention.
- Reaction to content.
- Community invitation or join request result.
- Event update or RSVP reminder.
- Collaboration request accepted.
- Chat message (when chat launches).

Requirements:

- Users can view, mark as read, and dismiss notifications.
- Users can configure notification preferences by category.
- Notification delivery should be asynchronous where appropriate.
- Avoid sending notifications for every low-value event; provide sensible defaults.

### 7.12 Servers, channels, and chat

**Priority:** P2

This module provides Discord-like community communication. It should be implemented after the core social and moderation foundations.

Requirements:

- Users can create or join chat servers, subject to server settings.
- Servers contain text channels with names, descriptions, and access rules.
- Server roles control who can create channels, invite members, and moderate messages.
- Support direct messages and group conversations.
- Support message history, timestamps, replies, reactions, and basic mentions.
- Support online/away/offline presence as an optional user-controlled status.
- WebSockets deliver new messages and presence updates in real time.
- Store messages persistently so users can retrieve history after reconnecting.
- Provide message reporting, user blocking, and moderator deletion.
- Add message rate limits and anti-spam controls.
- Voice/video channels are explicitly out of scope for this phase.

Acceptance criteria:

- Messages are persisted before or as part of successful delivery; reconnecting clients can load missed messages.
- Users cannot read channels they are not authorized to access.
- WebSocket connections authenticate and enforce the same permissions as REST endpoints.
- A disconnected client can reconnect and request messages after its last received message ID.

### 7.13 Friendship and interest-based discovery

**Priority:** P2

Requirements:

- Users can discover profiles based on shared anime, genres, interests, communities, and creator activities.
- Users can select whether they want to be discoverable.
- Users can express friendship interest or send a connection request.
- The recipient can accept, ignore, or block the request.
- Compatibility explanations must be understandable (for example, “You both like 8 of the same anime”), not presented as a scientifically validated score.
- Discovery should avoid revealing exact location or sensitive personal information.

Acceptance criteria:

- Discovery respects profile visibility and blocking preferences.
- Users can disable discovery without deleting their account.

### 7.14 Optional adult-only dating and matching

**Priority:** P3 — separate product gate; not part of MVP

Dating must be a distinct, opt-in feature rather than an automatic extension of ordinary fandom activity.

Requirements before launch:

- Independent dating opt-in and profile setup.
- Adults-only eligibility with a robust age-assurance approach appropriate to the operating jurisdictions; a self-entered birthday alone is not sufficient for a high-risk dating feature.
- Separate friendship and dating intent settings.
- Clear consent controls for visibility and matching.
- Matching only after mutual interest.
- Messaging unlocked only after a mutual match, with controls to unmatch, block, and report.
- No public display of exact location, live location, phone number, or private contact details by default.
- Strong moderation, abuse reporting, appeal handling, and safety education.
- Policies for impersonation, harassment, sexual harassment, scams, and unwanted contact.
- Clear separation between dating profiles and profiles of users who have not opted in.
- Legal/privacy review before release, including applicable age, privacy, consumer, and online safety requirements.

Acceptance criteria:

- Non-eligible users cannot access dating discovery or matching.
- Users who have not opted into dating are never displayed in dating discovery.
- A match requires positive action by both users.
- Either user can unmatch or block immediately.
- Reporting a dating-related safety issue enters a prioritized moderation workflow.

### 7.15 Moderation, reporting, and trust & safety

**Priority:** P0 (basic controls); P1/P2 (expanded workflow)

Requirements:

- Users can report posts, comments, profiles, communities, events, and messages when available.
- Report reasons include spam, harassment, impersonation, hate/abuse, inappropriate content, copyright concern, spoiler abuse, unsafe event, and other.
- Users can block other users.
- Community moderators can remove or lock community content and issue community-level warnings or restrictions.
- Platform moderators can review reports, remove content, suspend accounts, and record actions.
- Moderation actions must be logged with actor, target, reason, timestamp, and scope.
- Provide community rules and platform conduct guidelines.
- Establish an appeal process for significant platform enforcement actions.
- Add rate limits and spam prevention for account creation, posting, comments, messages, and invitations.
- Do not allow users to upload illegal content or content that violates platform rules. Provide a copyright reporting and takedown contact/process before accepting public media uploads.

MVP moderation minimum:

- Report content/account.
- Block user.
- Community owner/moderator role.
- Remove post/comment.
- Platform admin can review reports and suspend an account.
- Basic audit log for moderation actions.

### 7.16 Search and discovery

**Priority:** P0 (basic); P1 (filters and relevance)

Requirements:

- Search users, communities, posts, and events.
- Filter posts by anime title, content type, and recency.
- Filter communities by topic and popularity/activity.
- Filter events by date, type, and broad location.
- Use pagination and query limits.
- Search results must respect privacy and community permissions.

### 7.17 Account privacy and data controls

**Priority:** P0

Requirements:

- Users can set profile visibility.
- Users can control who can send direct messages when chat launches.
- Users can disable profile discovery.
- Users can mute or block other users.
- Users can review and manage active sessions where supported.
- Users can request account deletion.
- Public profile fields must be separated from private account/security fields.
- The platform must publish a privacy policy before public launch.

## 8. Key User Journeys

### 8.1 New user onboarding

1. Visitor opens the landing page.
2. User creates an account and verifies email.
3. User selects a username and adds a short profile description.
4. User selects favourite anime, genres, and interests (cosplay, fan art, manga, etc.).
5. User chooses profile visibility and notification defaults.
6. Platform suggests relevant communities and public events.
7. User follows communities or creators and lands on the Home feed.

**Success condition:** a new user can reach a relevant, personalized feed without being forced to complete an unnecessarily long profile.

### 8.2 Publish a community discussion

1. User opens a community.
2. User selects Create Post.
3. User chooses a post type and enters title/body.
4. User optionally tags anime and episode/chapter.
5. User marks spoilers where applicable.
6. User previews and publishes.
7. Other members can comment, vote/react, save, or report.

### 8.3 Share cosplay or crossplay content

1. User selects Create Post and chooses Creator/Cosplay content.
2. User uploads an image and adds a caption.
3. User adds optional character, series, event, and creator tags.
4. User reviews visibility and publishes.
5. Followers and relevant discovery surfaces can see the post according to ranking and privacy controls.

### 8.4 Create and attend an event

1. Organizer creates an event with date, timezone, type, description, and venue/online details.
2. Organizer selects visibility and optional capacity.
3. Event is published.
4. Fans discover it through Events, communities, or relevant tags.
5. Users RSVP and optionally join the event discussion.
6. Organizer posts updates and manages attendance.

### 8.5 Join a community

1. User discovers a community.
2. User views description, rules, recent posts, and member count.
3. User joins immediately if public or requests access if private.
4. User can post or comment according to the community's rules and permissions.

### 8.6 Future: send a direct message

1. User opens another user's public profile.
2. User selects Message if the recipient's privacy settings allow it.
3. The system checks blocks, permissions, and rate limits.
4. The conversation is created or opened.
5. Messages are delivered and persisted.

### 8.7 Future: dating match

1. An eligible adult explicitly opts into the separate dating feature.
2. User completes the required age-assurance and dating profile setup.
3. User chooses intent and discovery preferences.
4. User views eligible profiles and can like or pass.
5. A match is created only when interest is mutual.
6. Match-only messaging becomes available.
7. Either party can unmatch, block, or report.

## 9. MVP Scope and Release Plan

The product vision is broad, but the initial build must be deliberately constrained.

### Release 0 — Technical foundation

- Repository structure and coding conventions.
- Vue application shell and responsive navigation.
- Go application bootstrap and configuration.
- SQLite connection and migration runner.
- Dockerfiles for frontend and backend.
- Docker Compose for local development.
- Health endpoints and structured logs.
- CI checks and basic automated tests.

### Release 1 — Social foundation (MVP / first beta)

**Must ship:**

- Registration, login, email verification, logout, password reset.
- User profiles and anime-interest tags.
- Communities, public/private membership, owners, and moderators.
- Forum text/link/image posts.
- Comments and basic reactions or voting.
- Basic image-based social feed and following.
- Manual spoiler labels and spoiler reveal controls.
- Search across users, communities, and posts.
- In-app notifications for follows, replies, and mentions.
- Reporting, blocking, basic moderation queue, and admin controls.
- Responsive web design.
- Dockerized local and AWS deployment.

**Explicitly deferred:** real-time chat, video upload, voice/video, dating, sophisticated recommendation algorithms, full anime tracking, and advanced event operations.

### Release 2 — Creator and events

- Cosplay/crossplay/fan art categories.
- Creator profile highlights.
- Event creation and discovery.
- RSVP and event updates.
- Collaboration / Team Up requests.
- Event comments or linked event communities.
- Better location and date filters.

### Release 3 — Community communication

- Chat servers and text channels.
- Direct messages and group chats.
- WebSocket real-time message delivery.
- Presence and typing indicators (optional).
- Message history, replies, and reactions.
- Chat moderation and anti-spam controls.

### Release 4 — Interest discovery

- Friend discovery based on shared interests.
- Connection requests.
- Compatibility explanations.
- User-controlled discoverability.

### Release 5 — Optional adult dating

Proceed only after product validation, operational maturity, age assurance, legal review, safety policies, and moderation capacity are in place.

## 10. Technical Architecture

### 10.1 Architectural approach

Use a **modular monolith** for the initial backend. Keep modules separated by domain, but deploy one Go API process. This is easier to build, test, deploy, and operate than an early microservices architecture.

### 10.2 High-level components

```text
[Browser]
   |
   | HTTPS
   v
[Reverse Proxy / Static Web Server]
   |
   +---- serves Vue production assets
   |
   +---- /api/* and /ws/* ----------------------+
                                                |
                                                v
                                      [Go API Container]
                                                |
                                  +-------------+-------------+
                                  |                           |
                                  v                           v
                              [SQLite]                 [Object Storage]
                            EBS-backed file             Amazon S3
```

For the first AWS deployment, the reverse proxy, frontend static files, and Go API may run on one EC2 instance using Docker Compose. The frontend and backend should be separate images/containers. SQLite must live on a persistent EBS-backed host directory or volume, not in the disposable container filesystem.

### 10.3 Frontend

Recommended stack:

- Vue 3.
- TypeScript.
- Vite.
- Vue Router.
- Pinia for client state where needed.
- A consistent component and styling system (choose one approach and document it).
- WebSocket client for the later chat release.

Frontend responsibilities:

- Responsive UI and navigation.
- Form validation for user experience; backend remains the source of truth.
- Authentication state and route guards.
- Feed, community, event, profile, and chat screens.
- Loading, empty, error, and offline states.
- Accessible keyboard navigation and semantic markup.

### 10.4 Backend

Recommended stack:

- Go.
- Standard `net/http` or a small, well-supported router.
- SQL database access through a clearly defined repository layer.
- Database migrations.
- Structured logging.
- Request IDs and consistent API error responses.
- OpenAPI documentation for REST endpoints.
- Unit and integration tests.

Suggested module structure:

```text
backend/
  cmd/server/
  internal/
    auth/
    users/
    anime/
    communities/
    posts/
    comments/
    reactions/
    media/
    events/
    notifications/
    chat/          # later release
    discovery/     # later release
    matching/      # future, adults-only gate
    moderation/
    platform/
  migrations/
  Dockerfile
```

### 10.5 Database

Use SQLite for local development and the first single-instance deployment. Enable foreign keys and define indexes for frequently queried columns. Use transactions for operations such as membership creation, voting, RSVP capacity enforcement, and match creation if the dating module is ever introduced.

Potential initial entities:

- users
- user_profiles
- user_sessions or refresh_tokens (depending on auth model)
- anime_titles
- genres
- user_anime_interests
- user_interest_tags
- communities
- community_memberships
- community_roles
- community_rules
- posts
- post_media
- comments
- reactions or votes
- post_bookmarks
- follows
- notifications
- reports
- moderation_actions
- events (Release 2)
- event_rsvps (Release 2)
- collaboration_requests (Release 2)
- chat_servers, chat_channels, conversations, messages (Release 3)

Do not create all future tables upfront without a clear migration need. Add schema as features are implemented.

**SQLite deployment constraints:**

- Run one backend instance against the SQLite file.
- Do not mount the same SQLite database file across multiple independently writing application instances.
- Use a persistent EBS-backed path and test backup/restore.
- Use WAL mode only with a clear understanding of its file and backup requirements.
- Monitor database size, write contention, backup duration, and query latency.
- Plan PostgreSQL migration before horizontal scaling or high write concurrency.

### 10.6 Media storage

Use Amazon S3 for user-uploaded images when the AWS media feature is introduced.

- Store media metadata and object keys in SQLite; do not store binary image files in the database.
- Validate file extension, MIME type, actual file signature, file size, and image dimensions.
- Generate random object keys; do not use untrusted filenames as storage paths.
- Use presigned upload URLs or a controlled backend upload flow.
- Configure bucket access to block public access by default; serve public content through intentionally configured access paths/CDN later.
- Add image resizing/thumbnail generation when needed.
- Define retention and deletion rules for removed content and deleted accounts.

### 10.7 Real-time messaging (later release)

- Go WebSocket endpoint authenticated with the same account/session model.
- Channel membership and authorization checked server-side.
- Persistent message history in the database.
- Message IDs and cursor-based recovery after reconnect.
- Per-user and per-connection rate limits.
- Initial single-instance WebSocket hub is sufficient for learning and early use.
- Redis pub/sub or another shared messaging layer may be considered only when multiple backend instances are required.

## 11. AWS and Docker Deployment Requirements

### 11.1 Initial AWS architecture

| Component | Initial choice | Purpose |
|---|---|---|
| Compute | Amazon EC2 | Run Docker Compose and the application |
| Storage | Amazon EBS | Persist SQLite data and any required local operational data |
| Container registry | Amazon ECR | Store versioned frontend/backend images |
| Object storage | Amazon S3 | Store user-uploaded media |
| Identity/access | IAM | Least-privilege access for EC2 and deployment workflows |
| Networking | VPC + Security Groups | Isolate and control network access |
| Monitoring | CloudWatch | Collect logs and basic metrics/alarms |
| DNS | Route 53 (optional) | Manage a custom domain if used |
| TLS | Caddy or Nginx with an appropriate certificate workflow | Serve HTTPS |

Use the smallest suitable EC2 instance for a learning deployment and check current AWS pricing, free-tier eligibility, and regional availability before creating resources. Set a billing budget and alerts. Avoid leaving test resources running unnecessarily.

### 11.2 Docker requirements

- Separate frontend and backend Dockerfiles.
- Use multi-stage builds to keep runtime images small.
- Use non-root users where practical.
- Pin base image versions rather than relying on mutable `latest` tags.
- Use `.dockerignore` files.
- Do not bake secrets into images.
- Configure container health checks or equivalent health monitoring.
- Use Docker Compose for local development and the initial single-host deployment.
- Use named volumes or explicit bind mounts for persistent database data.
- Set CPU/memory limits where appropriate for the host.
- Ensure containers restart safely after host reboot.

### 11.3 CI/CD requirements

GitHub Actions pipeline:

1. Trigger on pull requests and pushes to the main branch.
2. Run Go formatting checks, static analysis, unit tests, and integration tests.
3. Run frontend lint/type checks and production build.
4. Build frontend and backend Docker images.
5. Scan dependencies and images using suitable security tooling.
6. On approved main-branch changes, publish versioned images to ECR.
7. Deploy to EC2 using a restricted deployment identity and a repeatable script.
8. Run health checks after deployment.
9. Keep the previous known-good image tag available for rollback.

Prefer short-lived AWS credentials through GitHub Actions OIDC instead of long-lived AWS access keys where supported by the chosen deployment setup.

### 11.4 Operations requirements

- HTTPS for all public traffic.
- Only required ports exposed publicly; SSH restricted to trusted addresses or replaced with a managed access method.
- SQLite database backups on a defined schedule.
- Backups stored separately from the EC2 instance (for example, encrypted S3 storage).
- At least one tested restore procedure.
- CloudWatch logs or another documented log destination.
- Health endpoint for application liveness/readiness.
- Basic alarms for EC2 status, disk capacity, memory where agent support exists, and application health.
- Documented deployment and rollback commands.
- Documented steps for rotating secrets and recovering from a failed deployment.

## 12. Non-Functional Requirements

### 12.1 Performance

Initial targets for the MVP under modest beta traffic:

- Common API reads should typically complete within 500 ms at the application layer under expected test load, excluding external provider or object-storage delays.
- Paginated feed responses should return a bounded number of records.
- Image thumbnails should be used in feeds instead of loading original large files.
- Avoid N+1 database query patterns.
- Use indexes for usernames, slugs, foreign keys, created timestamps, membership lookups, and common feed filters.

These are target thresholds to validate with load tests, not guaranteed production performance.

### 12.2 Availability and resilience

- Application startup should fail clearly when required configuration or database access is invalid.
- Temporary external metadata or object-storage failures should not take down unrelated community and profile functionality.
- Frontend should show clear errors and retry options for recoverable network failures.
- Database backup and restore must be documented and tested.
- Deployment should support rollback to a previous image version.

### 12.3 Security

- Enforce TLS in production.
- Hash passwords using a modern password-hashing function.
- Validate and normalize all input on the server.
- Use parameterized SQL queries.
- Enforce resource ownership and role permissions server-side.
- Protect against CSRF if cookie-based authentication is used.
- Apply rate limits to authentication, posting, comments, invites, uploads, and messaging.
- Limit upload sizes and types; inspect file signatures.
- Prevent path traversal and unsafe file handling.
- Keep secrets in a secure configuration mechanism and never commit them.
- Use least-privilege IAM policies and security groups.
- Audit privileged moderation and admin actions.
- Use dependency updates and vulnerability scanning.

### 12.4 Privacy

- Collect only information needed for product functionality.
- Do not expose a user's precise location by default.
- Provide profile visibility and discoverability controls.
- Avoid showing private email addresses or account security data in public APIs.
- Define data retention for deleted content, reports, logs, and backups.
- Provide a privacy policy and account deletion process before public launch.
- Dating data, if ever introduced, must have separate consent and access controls.

### 12.5 Accessibility and usability

- Responsive layouts for mobile, tablet, and desktop.
- Keyboard-operable navigation, menus, dialogs, and controls.
- Visible focus states.
- Semantic headings and form labels.
- Appropriate alt text for meaningful images.
- Sufficient text/background contrast.
- Avoid relying on colour alone to communicate moderation, spoiler, or event status.
- Include loading, empty, error, and success states for major views.

### 12.6 Maintainability

- Consistent formatting and linting.
- Clear module boundaries.
- Documented environment variables and setup steps.
- Database migration history committed to source control.
- Automated tests for core business rules and authorization.
- API documentation.
- ADRs (architecture decision records) for important technical choices such as authentication, SQLite-to-PostgreSQL migration, and media storage.

## 13. Analytics and Product Metrics

Track product usage in a privacy-conscious manner. Avoid collecting unnecessary personal information.

Suggested metrics:

| Metric | Definition | Why it matters |
|---|---|---|
| Activation rate | New users who follow/join at least one community or complete a meaningful action | Measures onboarding usefulness |
| Weekly active users | Unique users with meaningful activity in a 7-day period | Measures ongoing use |
| Contribution rate | Users who post, comment, react, or create an event | Measures community participation |
| Community health | Active communities and contributions per active community | Identifies empty or healthy spaces |
| Feed engagement | Meaningful interactions per feed impression | Helps evaluate discovery quality |
| Event conversion | Event page visitors who RSVP | Measures event usefulness |
| Creator retention | Creators who publish again within a defined period | Measures creator value |
| Report resolution time | Time between report creation and action/closure | Measures moderation operations |
| Deployment success rate | Successful deployments divided by total deployments | Measures delivery reliability |
| Recovery objective | Actual time to restore service/data in a recovery test | Measures operational readiness |

Do not optimize solely for time spent in the application or raw post volume. Community quality, safety, and user-reported usefulness matter.

## 14. API Design Outline

Use versioned REST endpoints, such as `/api/v1`. Exact payload schemas should be defined in OpenAPI as implementation begins.

### Authentication and users

```text
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
POST   /api/v1/auth/verify-email
POST   /api/v1/auth/password-reset/request
POST   /api/v1/auth/password-reset/complete
GET    /api/v1/me
PATCH  /api/v1/me
GET    /api/v1/users/{username}
POST   /api/v1/users/{id}/follow
DELETE /api/v1/users/{id}/follow
POST   /api/v1/users/{id}/block
DELETE /api/v1/users/{id}/block
```

### Communities

```text
GET    /api/v1/communities
POST   /api/v1/communities
GET    /api/v1/communities/{slug}
PATCH  /api/v1/communities/{slug}
POST   /api/v1/communities/{slug}/join
DELETE /api/v1/communities/{slug}/membership
GET    /api/v1/communities/{slug}/members
GET    /api/v1/communities/{slug}/posts
```

### Posts and comments

```text
GET    /api/v1/feed
POST   /api/v1/posts
GET    /api/v1/posts/{id}
PATCH  /api/v1/posts/{id}
DELETE /api/v1/posts/{id}
POST   /api/v1/posts/{id}/comments
GET    /api/v1/posts/{id}/comments
POST   /api/v1/posts/{id}/reactions
DELETE /api/v1/posts/{id}/reactions
POST   /api/v1/posts/{id}/bookmark
DELETE /api/v1/posts/{id}/bookmark
```

### Search, notifications, and moderation

```text
GET    /api/v1/search
GET    /api/v1/notifications
POST   /api/v1/notifications/{id}/read
POST   /api/v1/reports
GET    /api/v1/moderation/reports
POST   /api/v1/moderation/actions
```

### Events (Release 2)

```text
GET    /api/v1/events
POST   /api/v1/events
GET    /api/v1/events/{id}
PATCH  /api/v1/events/{id}
POST   /api/v1/events/{id}/rsvp
DELETE /api/v1/events/{id}/rsvp
POST   /api/v1/events/{id}/updates
```

### Chat (Release 3)

```text
GET    /api/v1/chat/servers
POST   /api/v1/chat/servers
GET    /api/v1/chat/servers/{id}/channels
POST   /api/v1/chat/servers/{id}/channels
GET    /api/v1/chat/conversations
GET    /api/v1/chat/conversations/{id}/messages
POST   /api/v1/chat/conversations/{id}/messages
WS     /api/v1/chat/ws
```

API requirements:

- Consistent JSON response format and error codes.
- Pagination parameters and stable ordering.
- Request validation and clear field-level validation errors.
- Authorization checks for every resource operation.
- Request IDs in logs and error responses.
- Rate-limit responses should be consistent.
- Avoid exposing internal database errors or stack traces.

## 15. Initial Data Model (Conceptual)

The following is a starting domain model, not a complete final schema.

| Entity | Important fields / relationships |
|---|---|
| User | id, email, password_hash, username, status, created_at |
| UserProfile | user_id, display_name, bio, avatar_key, banner_key, visibility, discoverable |
| AnimeTitle | id, external_id, title, alternate_title, media_type |
| UserAnimeInterest | user_id, anime_title_id, interest_type, sort_order |
| InterestTag | id, name, category |
| UserInterest | user_id, interest_tag_id |
| Community | id, owner_id, name, slug, description, visibility, rules |
| CommunityMembership | community_id, user_id, role, status, joined_at |
| Post | id, author_id, community_id nullable, type, title, body, spoiler_flag, created_at |
| PostMedia | id, post_id, storage_key, mime_type, size_bytes, width, height |
| Comment | id, post_id, author_id, parent_comment_id nullable, body, created_at |
| Reaction | id, user_id, target_type, target_id, reaction_type |
| Follow | follower_id, followed_id, created_at |
| Bookmark | user_id, post_id, created_at |
| Notification | id, user_id, type, actor_id, target_type, target_id, read_at |
| Report | id, reporter_id, target_type, target_id, reason, details, status |
| ModerationAction | id, moderator_id, target_type, target_id, action, reason, created_at |
| Event (Release 2) | id, organizer_id, title, description, event_type, starts_at, ends_at, timezone, venue_text |
| EventRSVP (Release 2) | event_id, user_id, status, created_at |

Schema guidelines:

- Use immutable IDs rather than usernames as foreign keys.
- Enforce unique constraints for usernames, community slugs, and one-per-user reactions/votes where appropriate.
- Add created/updated timestamps consistently.
- Define delete behavior carefully; avoid cascading deletion of records that must be retained for moderation, audit, or legal obligations.
- Store event times with an explicit timezone strategy.
- Use a consistent soft-delete policy where content removal needs to preserve thread integrity.

## 16. UX and Visual Design Direction

### 16.1 Overall direction

A modern, polished social platform with anime culture as its subject matter—not a cluttered fan wiki and not a generic admin dashboard.

Suggested direction:

- Dark theme as the initial visual default, with a considered light theme later.
- Deep charcoal/ink surfaces, layered cards, subtle borders, and restrained accent colours.
- Anime artwork should be user-generated, licensed, or sourced through permitted metadata/image providers.
- Use typography that is readable for long discussions.
- Use expressive visual treatments for events and creator content without making the forum difficult to scan.
- Keep the main navigation consistent across modules.
- Avoid excessive gradients, random anime character imagery, and generic AI-generated branding.
- Express the PromisedLand identity through a sense of belonging and discovery; do not rely on generic anime clichés or copyrighted franchise characters as the platform logo.

### 16.2 Important screens

MVP screens:

1. Landing page.
2. Register, login, verification, password reset.
3. Onboarding and interest selection.
4. Home feed.
5. Community discovery.
6. Community detail and rules.
7. Post detail and comment thread.
8. Create/edit post.
9. User profile.
10. Search results.
11. Notifications.
12. Report flow.
13. Moderator queue.
14. Settings and privacy.

Later screens:

- Event discovery and event details.
- Event creation and management.
- Creator portfolio.
- Collaboration board.
- Chat server/channel layout.
- Direct messages.
- Friend discovery.
- Adults-only dating discovery and match inbox (separate gated experience).

### 16.3 UX states

Every major list or data view should define:

- Loading state.
- Empty state with a useful next action.
- Error state with retry guidance.
- Success feedback after user actions.
- Permission-denied state.
- Offline or reconnecting state where real-time features exist.
- Deleted or unavailable content state.

## 17. Testing Strategy

### 17.1 Backend tests

- Unit tests for validation and business rules.
- Integration tests against a temporary SQLite database.
- Authorization tests for owner, member, moderator, and visitor access.
- Tests for private community access.
- Tests for spoiler metadata handling.
- Tests for duplicate reactions/votes and transactional operations.
- Tests for upload validation and access control.
- API contract tests based on OpenAPI where practical.

### 17.2 Frontend tests

- Component tests for forms and interactive components.
- Route and authentication-guard tests.
- Tests for hidden/revealed spoiler states.
- Tests for feed pagination and error states.
- Accessibility checks for key screens.
- End-to-end tests for registration, joining a community, publishing a post, commenting, and reporting.

### 17.3 Infrastructure tests

- Build both Docker images from a clean checkout.
- Start the application using Docker Compose.
- Verify frontend-to-backend networking.
- Recreate backend container and confirm SQLite data persists.
- Reboot test EC2 instance and verify services recover.
- Verify S3 upload permissions and denied access cases.
- Verify HTTPS and security-group exposure.
- Test backup restoration to a clean environment.
- Test deployment rollback.

## 18. Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Scope becomes too large | Project may never reach a usable release | Keep Release 1 focused; defer chat and dating |
| Empty communities | Users see little value | Seed a few focused communities; invite clubs and friends; support useful onboarding |
| Moderation workload | Harassment, spam, unsafe content | Clear rules, reports, blocking, moderation tools, rate limits, community roles |
| Spoilers leak | Poor experience and loss of trust | Manual spoiler labels first; episode-aware rules and reporting |
| Copyright complaints | Takedowns and legal risk | User content rules, reporting, takedown workflow, no anime episode hosting |
| Image storage costs | Unexpected AWS bill | File-size limits, thumbnailing, quotas, lifecycle rules, budget alerts |
| SQLite write contention | Slow writes or failures under growth | Single instance initially; monitor contention; plan PostgreSQL migration |
| WebSocket complexity | Reliability and scaling challenges | Defer chat; start with a single-instance hub; add shared pub/sub only when needed |
| Dating safety risks | Serious user harm and legal exposure | Adults-only gate, age assurance, opt-in, privacy by design, reporting, review before launch |
| External anime API dependency | Metadata outages or policy changes | Cache allowed data, isolate provider integration, allow manual title suggestions |
| Cloud spend | Unplanned cost | Budget alerts, small initial resources, resource teardown checklist |

## 19. Dependencies and External Services

Potential integrations, subject to current terms, availability, and suitability:

- Anime metadata provider for titles, seasons, episodes, and artwork.
- Amazon S3 for uploaded media.
- Email delivery provider for verification and password-reset messages.
- Optional image processing service or application-side image library.
- Optional analytics provider with privacy-conscious configuration.

Do not assume an external provider permits permanent caching, redistribution of artwork, or commercial use. Review each provider's terms before integration. Core community functionality should continue if the anime metadata provider is unavailable.

## 20. Open Product Decisions

These decisions should be made before or during implementation; defaults below are suggested for the first version.

| Decision | Suggested initial default |
|---|---|
| Product name | PromisedLand (working brand; verify trademark, company-name, social-handle, and domain availability before public branding) |
| Initial audience | Anime fans, college clubs, cosplay groups, and online fandom communities |
| Initial geography | Build globally usable; seed and validate with a small initial community |
| Primary experience | Community-first, with a social feed integrated into the platform |
| Feed ranking | Following + followed communities + relevant tags + recency; no ML initially |
| Community visibility | Public and private |
| Profile visibility | Public/private setting, with discoverability separately configurable |
| Media in MVP | Images only; video deferred |
| Chat in MVP | No; Release 3 |
| Events in MVP | Basic event posts may be deferred; full RSVP and event management in Release 2 |
| Dating | Not in MVP; adults-only, opt-in future release |
| Database | SQLite for one-instance learning deployment; PostgreSQL when scaling requirements justify it |
| Media storage | S3 for production uploads; local filesystem or MinIO for local development if useful |
| Hosting | One EC2 host with Docker Compose initially |
| Frontend deployment | Static Vue build served by a small web server/reverse proxy container |
| Backend deployment | One Go API container |

## 21. Definition of Done

A feature is complete when:

- The feature's acceptance criteria are met.
- Backend validation and authorization are implemented.
- Frontend loading, empty, error, and success states are handled.
- Unit/integration tests cover important business rules.
- Responsive behavior is verified.
- Accessibility basics are checked.
- Logs do not expose passwords, tokens, private message content, or sensitive profile data unnecessarily.
- Database migrations are committed and tested.
- API documentation is updated.
- Docker build and local Compose startup succeed.
- Deployment configuration and environment variables are documented.
- Relevant privacy and moderation implications are addressed.

## 22. Suggested Repository Layout

A single repository is recommended for the initial project.

```text
promisedland/
├── frontend/
│   ├── src/
│   ├── public/
│   ├── Dockerfile
│   └── package.json
├── backend/
│   ├── cmd/
│   ├── internal/
│   ├── migrations/
│   ├── Dockerfile
│   └── go.mod
├── deploy/
│   ├── nginx/                 # or Caddy configuration
│   ├── compose/
│   ├── scripts/
│   └── aws/                   # deployment notes / IaC when introduced
├── docs/
│   ├── architecture.md
│   ├── api.md
│   ├── threat-model.md
│   └── decisions/
├── .github/
│   └── workflows/
├── docker-compose.yml
├── .env.example
├── .gitignore
├── PRD.md
└── README.md
```

## 23. Final Product Recommendation

Build PromisedLand as a **community-first anime social network** anchored in the brand promise: **Your fandom. Your people. Your place.**

The first public release should focus on the parts that make the platform useful without requiring a large real-time infrastructure investment:

- Profiles and anime interests.
- Communities and moderated discussions.
- A social feed for fan art, cosplay, crossplay, and anime-related posts.
- Spoiler-aware content controls.
- Following, search, and notifications.
- Basic safety and moderation tools.

Then add events and collaboration, followed by Discord-like chat. Treat friendship discovery and dating as separate later product decisions; do not let dating scope delay the core platform.

For the portfolio, document the engineering decisions and show the running product, Docker images, AWS architecture, CI/CD workflow, monitoring, and a tested backup/restore process. Those operational details will make the project much more credible than feature count alone.

---

**End of PRD — PromisedLand v1.1**
