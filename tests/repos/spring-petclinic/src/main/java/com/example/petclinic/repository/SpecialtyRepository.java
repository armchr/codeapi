package com.example.petclinic.repository;

import com.example.petclinic.model.Specialty;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;
import java.util.Optional;

/**
 * Repository for Specialty entity.
 */
@Repository
public interface SpecialtyRepository extends JpaRepository<Specialty, Long> {

    /**
     * Find specialty by name.
     */
    Optional<Specialty> findByName(String name);

    /**
     * Check if specialty exists by name.
     */
    boolean existsByName(String name);
}
