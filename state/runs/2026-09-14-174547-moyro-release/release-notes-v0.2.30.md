moyro v0.2.30

Visitors who already hold a Keycloak session can now be signed in without
seeing the login screen. A new `auto_login` switch on the Keycloak settings,
off by default, lets the web app ask the provider once with `prompt=none`;
a refusal such as `login_required` sends the browser to `/login?sso=none`
with its deep link preserved instead of surfacing an error, the attempt is
remembered for the browser session so a refusal or sign-out can never loop,
and an ordinary login flow keeps reporting the same provider errors as
before.
