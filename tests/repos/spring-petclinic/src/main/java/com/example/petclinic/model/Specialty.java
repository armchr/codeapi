package com.example.petclinic.model;

import jakarta.persistence.*;
import jakarta.validation.constraints.NotBlank;

/**
 * Entity representing a veterinary specialty (e.g., surgery, dentistry).
 */
@Entity
@Table(name = "specialties")
public class Specialty extends BaseEntity {

    @NotBlank(message = "Specialty name is required")
    @Column(name = "name")
    private String name;

    public Specialty() {
    }

    public Specialty(String name) {
        this.name = name;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }
}
