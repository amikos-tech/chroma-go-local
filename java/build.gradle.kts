plugins {
    base
}

allprojects {
    group = "tech.amikos"
    version = "0.3.1-SNAPSHOT"

    repositories {
        mavenCentral()
    }
}

subprojects {
    tasks.withType<Test>().configureEach {
        useJUnitPlatform()
        val sharedLib = System.getenv("CHROMA_LIB_PATH") ?: ""
        if (sharedLib.isNotBlank()) {
            environment("CHROMA_LIB_PATH", sharedLib)
        }
    }
}
