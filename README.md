# TL;DR

The reason I created __TL;DR__ was to check if I could decouple the _front_ from the _backend_.
This is just a little _poc_, nothing serious and it was created within a work-week on my break time, so don't look to close into it :>

## Tech-Stack

### Frontend

The frontend is nothing stylish, it is written in _javascript_, uses the _expressjs_ framework (_as the server_) and axios for fetching data from the __api__. For displaying content I use _jade_ as the templating and rendering engine.

### Backend

For the backend I choose _golang_ as the language and _fiber_ framework to handle all the server stuff (_get_/_post_ and all that jazz). The data is stored in an _sqlite_ database so we don't loose anything important.
