package com.example.petclinic.model;

import java.util.HashSet;
import java.util.Set;

/**
 * Data Transfer Object for Vet entity.
 */
public class VetDto {
    private Long id;
    private String firstName;
    private String lastName;
    private Set<String> specialties = new HashSet<>();

    public VetDto() {}

    public VetDto(Long id, String firstName, String lastName) {
        this.id = id;
        this.firstName = firstName;
        this.lastName = lastName;
    }

    // Getters and Setters
    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }

    public String getFirstName() { return firstName; }
    public void setFirstName(String firstName) { this.firstName = firstName; }

    public String getLastName() { return lastName; }
    public void setLastName(String lastName) { this.lastName = lastName; }

    public String getFullName() {
        return (firstName != null ? firstName : "") + " " + (lastName != null ? lastName : "");
    }

    public Set<String> getSpecialties() { return specialties; }
    public void setSpecialties(Set<String> specialties) {
        this.specialties = specialties != null ? specialties : new HashSet<>();
    }

    public String getSpecialtiesDisplay() {
        if (specialties == null || specialties.isEmpty()) {
            return "None";
        }
        return String.join(", ", specialties);
    }
}
