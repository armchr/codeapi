package com.example.petclinic;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.cache.annotation.EnableCaching;
import org.springframework.scheduling.annotation.EnableAsync;

/**
 * Main application class for the Pet Clinic Spring Boot application.
 * Demonstrates Spring Boot auto-configuration, caching, and async processing.
 */
@SpringBootApplication
@EnableCaching
@EnableAsync
public class PetClinicApplication {

    public static void main(String[] args) {
        SpringApplication.run(PetClinicApplication.class, args);
    }
}
