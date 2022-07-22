#### SAG bot

Custom solution to handle banning players upon death. Scans Minecraft pipe file for player deaths and stores their death as a Unix timestamp. Every second it checks to see if this timestamp expires then sends an unban command afterward. 

Ideally this should be implemented as a fabric mod (as we are using a fabric server) but this is very portable and I refuse to touch icky Java. 

This is not designed for other people to use it as it's horribly coded and undocumented, but if you have any issues you can contact me.
