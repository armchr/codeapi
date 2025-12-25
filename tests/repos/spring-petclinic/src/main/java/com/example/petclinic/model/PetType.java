package com.example.petclinic.model;

import jakarta.persistence.*;
import jakarta.validation.constraints.NotBlank;

/**
 * Entity representing a type of pet (e.g., dog, cat, bird).
 */
@Entity
@Table(name = "pet_types")
public class PetType extends BaseEntity {

    @NotBlank(message = "Pet type name is required")
    @Column(name = "name")
    private String name;

    public PetType() {
    }

    public PetType(String name) {
        this.name = name;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    @Override
    public String toString() {
        return name;
    }
}
