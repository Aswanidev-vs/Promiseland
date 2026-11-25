<script setup lang="ts">
import LegalPage from '@/components/layout/LegalPage.vue'
</script>

<template>
  <LegalPage
    title="Privacy policy"
    updated="25 November 2026"
    intro="What PromisedLand collects, why, and what you can do about it. This is a working draft for the closed beta: it has not had legal review and will change before launch."
  >
    <h2>1. What we collect</h2>
    <ul>
      <li>
        <strong>Account details:</strong> email address, username, display name, and your password
        stored only as a salted hash — we never keep the password itself.
      </li>
      <li>
        <strong>Profile details you add:</strong> bio, avatar, banner, the anime titles and crafts
        you pick as interests, and the visibility settings you choose.
      </li>
      <li>
        <strong>What you do here:</strong> posts, comments, votes, reactions, event RSVPs, team-up
        applications, and chat messages when that release is live.
      </li>
      <li>
        <strong>Technical basics:</strong> a session cookie, request logs with a request id, and
        error reports needed to keep the service running and secure.
      </li>
    </ul>

    <h2>2. What we do not do</h2>
    <p>
      PromisedLand does not sell personal data, does not run third-party advertising, and does not
      embed ad-network trackers. The product is meant to be paid for by the people using it, not by
      attention brokering.
    </p>

    <h2>3. Why we use it</h2>
    <ul>
      <li>To sign you in, verify your email, and recover your account.</li>
      <li>To show you communities, feeds and events that match the interests you chose.</li>
      <li>To keep the platform safe: moderation, reports, blocking, and abuse prevention.</li>
      <li>To know when something is broken, via error logs rather than behavioural profiling.</li>
    </ul>

    <h2>4. Cookies and local storage</h2>
    <p>
      We set one session cookie so you stay signed in. In your browser's local storage we keep your
      theme choice and whether you chose to browse as a guest. No advertising or cross-site tracking
      cookies are used.
    </p>

    <h2>5. What is public</h2>
    <p>
      Posts, comments, profiles and events can be public on PromisedLand, and public pages can be
      read by visitors without an account. Your settings control profile visibility, what appears in
      discovery, and who can message you. Treat anything you post publicly as something a stranger
      can read, copy, or screenshot.
    </p>

    <h2>6. Who we share it with</h2>
    <p>
      We use infrastructure providers to host the app and its database, and those providers process
      data on our instructions. We may also share information where the law requires it, or where it
      is necessary to stop serious harm. We do not sell or rent personal data to anyone.
    </p>

    <h2>7. How long we keep it</h2>
    <p>
      Account data stays while your account exists. Deleting your account removes your profile and
      your content from public view; some records — moderation decisions, security logs and backups —
      may be kept for a limited period where we need them for safety or legal reasons.
    </p>

    <h2>8. Your choices</h2>
    <ul>
      <li>Edit or delete almost anything from your profile and settings pages.</li>
      <li>Change who can find you, follow you, and message you.</li>
      <li>Ask for a copy of your account data, or for the account to be closed, from settings.</li>
    </ul>
    <p>
      Where local law gives you additional rights over your data, those rights apply on top of these
      controls.
    </p>

    <h2>9. Age</h2>
    <p>
      PromisedLand is not for children under 13, and the adults-only area is switched off during this
      beta. If we learn that an account belongs to someone under 13, we will close it.
    </p>

    <h2>10. Security</h2>
    <p>
      Passwords are hashed, sessions travel over HTTPS, and access to production data is limited to
      the people running the service. No system is perfect: if a breach ever affects your account, we
      will tell you what happened and what to do.
    </p>

    <h2>11. Changes and contact</h2>
    <p>
      If this policy changes in a way that matters, we will say so in the app. Questions about your
      data: use the report form in the app, or reply to the verification email your account was
      created with.
    </p>

    <p>
      See also the <RouterLink to="/terms">terms of use</RouterLink>.
    </p>
  </LegalPage>
</template>
