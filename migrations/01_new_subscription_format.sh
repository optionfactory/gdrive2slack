#!/bin/bash


# with_entries(fn) -> to_entries | map(fn) | from_entries
# 		where entries are [{key: K, value:V}, ...]

cat $1 | jq  '
	with_entries(
		.value = {
			channel: .value.channel, 
			slack_access_token: .value.soauth.access_token, 
			google_refresh_token: .value.goauth.refresh_token, 
			guser: {
				displayName: .value.guser.displayName,
      			givenName: .value.guser.name.givenName,
      			familyName: .value.guser.name.familyName,
      			email: .value.guser.emails[0].value
			},
			suser: .value.suser
		}
	)'