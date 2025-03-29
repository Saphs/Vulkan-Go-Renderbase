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

## Low amount of libraries

One of the goals of this project is to write a renderer from scratch, with as little help as possible. This will allow
for total control over every aspect of the software without being held to decisions made by libraries. It also provides
a great learning environment to truly understand what goes into user interaction and rendering.

### Math module

Any renderer will use some sort of mathematical functionality to represent vectors, matrices and utility functions. To
avoid a library, a vector_math module was written. See: [vector_math](/vector_math/README.md) for more information.

### Abstraction

Since Vulkan is very explicit, reusing and abstracting certain functions is a necessity. This is done inside the common
module. It is structured into 4 types of structs and functions:
1. **Wrapper**: A function or struct that wraps a raw vulkan entity without altering any behavior or arguments with the
sole purpose of providing a neater, go-like signature. See: [vk_wrappers](/common/vk_wrappers.go) as an example.
2. **Simplification**: A function or struct setting sensible default values that will be correct in most cases but will
slightly alter the interface to accommodate a sleeker look. These are things like removing a parameter that specifies
the size of a given array by calculating it and providing it implicitly to the underlying vulkan function. See 
[vk_simplifications](/common/vk_simplifications.go) for examples.
3. **Abstraction**: A function or struct building common function calls from raw Vulkan calls, Wrappers or Simplifications
to provide convenience for things done often. This should strife to keep most of the underlying flexibility but will
inevitably hide some functionality by defaulting values of the underlying functions or groups them to achieve a bigger
action. See: [vk_abstraction](/common/vk_abstraction.go) for examples.
4. **Utility classes**: These are structs building specific components used by the renderer module. This includes things
like a window class bundling all window related things coming from SDL and Vulkan to provide a simpler time talking to
this sub-category of calls. They serve mostly as a logical grouping of raw access functionality. This allows for a neater
renderer Core that then orchestrates these utility objects. See: [vk_sdl_window](/common/vk_sdl_window.go) as an example.

---

## Screenshots

![29-03-2025](/doc/29-03-2025_progress.png)