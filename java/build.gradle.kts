import org.gradle.jvm.tasks.Jar

plugins {
    base
}

val defaultSnapshotVersion = "0.0.0-SNAPSHOT"

fun normalizeReleaseVersion(raw: String): String {
    val trimmed = raw.trim()
    if (trimmed.isEmpty()) {
        return defaultSnapshotVersion
    }

    return trimmed
        .removePrefix("refs/tags/")
        .removePrefix("v")
}

val javaArtifactVersion = providers.gradleProperty("releaseVersion")
    .orElse(providers.environmentVariable("RELEASE_VERSION"))
    .map(::normalizeReleaseVersion)
    .orElse(defaultSnapshotVersion)

allprojects {
    group = "tech.amikos"
    version = javaArtifactVersion.get()

    repositories {
        mavenCentral()
    }
}

subprojects {
    tasks.withType<Jar>().configureEach {
        manifest {
            attributes(
                "Implementation-Title" to "chroma-local-java-${project.name}",
                "Implementation-Version" to project.version.toString()
            )
        }
    }

    tasks.withType<Test>().configureEach {
        useJUnitPlatform()
        val sharedLib = System.getenv("CHROMA_LIB_PATH") ?: ""
        if (sharedLib.isNotBlank()) {
            environment("CHROMA_LIB_PATH", sharedLib)
        }
    }
}
