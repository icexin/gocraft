#version 330 core

in vec3 pos;

uniform mat4 matrix;

void main() {
    gl_Position = matrix *  vec4(pos, 1.0);
}
