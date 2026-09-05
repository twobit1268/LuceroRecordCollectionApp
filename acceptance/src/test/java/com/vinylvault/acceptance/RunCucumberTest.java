package com.vinylvault.acceptance;

import org.junit.platform.suite.api.IncludeEngines;
import org.junit.platform.suite.api.SelectClasspathResource;
import org.junit.platform.suite.api.Suite;

/**
 * Single entry point for the acceptance suite.
 *
 * <p>{@code mvn test} (via Surefire), IntelliJ, and CI all run this class. The
 * {@code cucumber} JUnit Platform engine then discovers every {@code .feature}
 * file on the classpath under {@code com/vinylvault/acceptance} and executes it
 * using the step definitions in that same package (configured as the glue path
 * in {@code src/test/resources/junit-platform.properties}).
 *
 * <p>There is deliberately no code here - this is the modern replacement for the
 * old {@code @RunWith(Cucumber.class)} runner.
 */
@Suite
@IncludeEngines("cucumber")
@SelectClasspathResource("com/vinylvault/acceptance")
public class RunCucumberTest {
}
