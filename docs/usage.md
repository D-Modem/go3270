# Using go3270

**Project status:** This library has been used by a small number of projects, and I believe the overall functionality is sound and relatively bug-free. Feedback is appreciated. At this point, I will try not to make breaking changes to the API, but I have not yet declared the library to be at v1.0, so I make no promises. (upstream's note)

See the example folders for quick demonstrations of using the library:

 * example1 uses the lower-level function ShowScreenOpts().
 * example2 uses a higher-level convenience function HandleScreen().
 * example3 demonstrates updating the client's 3270 display while waiting for a response by using an update thread and a waiting response thread.
 * example4 demonstrates the RunTransactions() approach to handing control from one screen to the next. This is the recommended way to build applications using go3270.
 * example5 demonstrates support for larger-than-default (24x80) terminal sizes.

For larger applications, I recommend using the `RunTransactions()` function to serve as the driver for your application. You can implement transaction functions which pass control from one transaction to another. example4 demonstrates a "larger" application that uses this approach.

For an example of a complete application that handles multiple user sessions, adapts to different screen sizes, and with a nice written explanation of how the application is built, Moshix has created a [Minesweeper game][moshix-minesweeper] using go3270 to serve as a resource for the community to learn from.

[moshix-minesweeper]: https://github.com/moshix/minesweeper

Here's [a video introducing the library][introVideo] as well.

[introVideo]: https://www.youtube.com/watch?v=h9XTjup5W5U
