# Golang based Vulkan render base application
This repository aims to provide a jumping off point for personal projects of mine.
It was written on windows but only uses platform independent libraries and should thus be platform independent.
Some compatability issues may exist but should be fairly simple to resolve.

## Basic structure
The application base is as minimalistic as possible and only uses go bindings for
[SDL2](https://github.com/veandco/go-sdl2) and [Vulkan](https://github.com/goki/vulkan) as its libraries.
![Basic structure](/doc/VulkanRenderbaseStructure.drawio.png)

SDL2 is used to interact with both the user input events and handling the OS specific window management. Vulkan is
then initialized to render directly into the window provided by SDL2.