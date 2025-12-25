package com.example.petclinic.repository;

import com.example.petclinic.model.Visit;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;
import java.time.LocalDate;
import java.util.List;

/**
 * Repository for Visit entity.
 */
@Repository
public interface VisitRepository extends JpaRepository<Visit, Long> {

    /**
     * Find visits by pet ID.
     */
    List<Visit> findByPetId(Long petId);

    /**
     * Find visits by vet ID.
     */
    List<Visit> findByVetId(Long vetId);

    /**
     * Find visits between dates.
     */
    List<Visit> findByDateBetween(LocalDate startDate, LocalDate endDate);

    /**
     * Find recent visits for a pet.
     */
    @Query("SELECT v FROM Visit v WHERE v.pet.id = :petId ORDER BY v.date DESC")
    List<Visit> findRecentVisitsByPetId(@Param("petId") Long petId);

    /**
     * Find visits by pet owner.
     */
    @Query("SELECT v FROM Visit v WHERE v.pet.owner.id = :ownerId ORDER BY v.date DESC")
    List<Visit> findByOwnerId(@Param("ownerId") Long ownerId);

    /**
     * Count visits for a pet.
     */
    long countByPetId(Long petId);

    /**
     * Find today's visits.
     */
    @Query("SELECT v FROM Visit v WHERE v.date = :today")
    List<Visit> findTodaysVisits(@Param("today") LocalDate today);
}
