# What is this for?
I'm a big fan of review deployment! 
So I want to deploy each merge/pull request on kubernetes to test the feature. 

Now, I have a problem with OAuth and the redirect uri  if I do a domain-based distinction (like: mr-1.my-awesome-domain.com and mr-2.my-awesome-domain.com). 
The easy solution would be to add the new redirect to the OAuth server of choice and call it a day. 

There are situations where this is not possible because you are not allowed to access the server programmatically or there is no way to do this (looking at you, Google). 
So the idea here is to create a server in between with redirects to your review deployment! 
