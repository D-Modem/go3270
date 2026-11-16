# Using go3270

**Project status:** This library has been used by a small number of projects, and I believe the overall functionality is sound and relatively bug-free. Feedback is appreciated. At this point, I will try not to make breaking changes to the API, but I have not yet declared the library to be at v1.0, so I make no promises. (upstream's note)

The example folders show quick demonstrations of the library:

 * example1 uses the lower-level function ShowScreenOpts().
 * example2 uses a higher-level convenience function HandleScreen().
 * example3 shows how to update the 3270 display of the client while the server waits for a response. It uses an update thread and a thread that waits for the response.
 * example4 shows the RunTransactions() method, which gives control from one screen to the next. This is the recommended method to build applications with go3270.
 * example5 shows the support for terminal sizes that are larger than the default (24x80).

For larger applications, use the `RunTransactions()` function as the driver of your application. You can write transaction functions that give control from one transaction to another. example4 shows a "larger" application that uses this method.

Moshix made a [Minesweeper game][moshix-minesweeper] with go3270, as a resource for the community. It is an example of a complete application. It controls many user sessions and adapts to different screen sizes. It also has a good written explanation of how the application is built.

[moshix-minesweeper]: https://github.com/moshix/minesweeper

There is also [a video that introduces the library][introVideo].

[introVideo]: https://www.youtube.com/watch?v=h9XTjup5W5U
