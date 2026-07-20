#version 330 core

uniform float time;

out vec4 color;

void main() {
    float pulse = sin(time * 4.0) * 0.3 + 0.5;
    color = vec4(pulse, pulse, pulse, 1.0);
}
