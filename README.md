# Collector

This is a small web app to collect training data by presenting the user some
input data, and waiting for them to provide the output. It only supports a
single type of each, right now, but I have vague plans to make it flexible.

I'm trying to use this to generate training data which is suitable for my very
simple control model in [rl-sandbox][]. I want a robot to move towards the red
box, but when I manually capture training data from the actual pixels (via my
eyeballs) using a simulator, it barely works at all, because at runtime the
pixels first pass through a vision model. This program is intended to present
me with the same input data that the control model gets, so I can provide
examples based only on that.

## License

MIT

[rl-sandbox]: https://github.com/adammck/rl-sandbox
