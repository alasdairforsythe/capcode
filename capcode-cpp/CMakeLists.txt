cmake_minimum_required(VERSION 3.20)
project(capcode_cpp VERSION 0.1.0 LANGUAGES CXX)

include(CMakePackageConfigHelpers)
include(GNUInstallDirs)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

option(CAPCODE_CPP_BUILD_TESTS "Build capcode-cpp tests" ${PROJECT_IS_TOP_LEVEL})

find_package(PkgConfig REQUIRED)
pkg_check_modules(ICU REQUIRED IMPORTED_TARGET icu-uc)

add_library(capcode_cpp
  src/capcode.cpp
)
add_library(capcode::capcode ALIAS capcode_cpp)
set_target_properties(capcode_cpp PROPERTIES EXPORT_NAME capcode)

target_include_directories(capcode_cpp
  PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:${CMAKE_INSTALL_INCLUDEDIR}>
)

target_link_libraries(capcode_cpp
  PRIVATE
    PkgConfig::ICU
)

target_compile_features(capcode_cpp PUBLIC cxx_std_20)

if(CMAKE_CXX_COMPILER_ID MATCHES "Clang|GNU")
  target_compile_options(capcode_cpp PRIVATE -Wall -Wextra -Wpedantic)
endif()

install(TARGETS capcode_cpp
  EXPORT capcode_cppTargets
  ARCHIVE DESTINATION ${CMAKE_INSTALL_LIBDIR}
  LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR}
  RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR}
  INCLUDES DESTINATION ${CMAKE_INSTALL_INCLUDEDIR}
)
install(DIRECTORY include/ DESTINATION ${CMAKE_INSTALL_INCLUDEDIR})

write_basic_package_version_file(
  "${CMAKE_CURRENT_BINARY_DIR}/capcode_cppConfigVersion.cmake"
  VERSION ${PROJECT_VERSION}
  COMPATIBILITY SameMajorVersion
)

configure_package_config_file(
  cmake/capcode_cppConfig.cmake.in
  "${CMAKE_CURRENT_BINARY_DIR}/capcode_cppConfig.cmake"
  INSTALL_DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/capcode_cpp
)

install(FILES
  "${CMAKE_CURRENT_BINARY_DIR}/capcode_cppConfig.cmake"
  "${CMAKE_CURRENT_BINARY_DIR}/capcode_cppConfigVersion.cmake"
  DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/capcode_cpp
)
install(EXPORT capcode_cppTargets
  NAMESPACE capcode::
  DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/capcode_cpp
)

if(CAPCODE_CPP_BUILD_TESTS)
  enable_testing()
  add_executable(capcode_unit tests/capcode_unit.cpp)
  target_link_libraries(capcode_unit PRIVATE capcode_cpp)
  add_test(NAME capcode_unit COMMAND capcode_unit)
endif()
